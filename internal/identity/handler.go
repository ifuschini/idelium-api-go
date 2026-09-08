// Package identity contains fail-closed gates for late-wave identity migration.
package identity

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/idelium/idelium-api-go/internal/browserauth"
	"github.com/idelium/idelium-api-go/internal/httpx"
)

var replayMu sync.Mutex
var replayedNonces = map[string]time.Time{}

// Handler exposes advanced identity routes only after migration gates enable a
// Go-native implementation. Until then it fails closed with safe diagnostics.
type Handler struct {
	logger    *slog.Logger
	sessions  browserauth.SessionRepository
	providers ProviderRepository
	mfa       MFARepository
	scim      SCIMRepository
	users     browserauth.UserRepository
}

type Provider struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Issuer   string `json:"issuer,omitempty"`
	Audience string `json:"audience,omitempty"`
	Status   string `json:"status"`
}
type ProviderRepository interface {
	ListProviders(context.Context, int64) ([]Provider, error)
	CreateProvider(context.Context, int64, Provider) (Provider, error)
}
type MFARepository interface {
	SetMFASecret(context.Context, int64, string) error
	GetMFASecret(context.Context, int64) (string, error)
	MarkMFAConfirmed(context.Context, int64, time.Time) error
}
type SCIMRepository interface {
	CreateSCIMUser(context.Context, int64, string, string, bool) (browserauth.User, error)
}

// NewHandler creates an advanced identity migration gate.
func NewHandler(logger *slog.Logger, deps ...any) Handler {
	h := Handler{logger: logger}
	if len(deps) > 0 {
		h.sessions, _ = deps[0].(browserauth.SessionRepository)
	}
	if len(deps) > 1 {
		h.providers, _ = deps[1].(ProviderRepository)
	}
	if len(deps) > 2 {
		h.mfa, _ = deps[2].(MFARepository)
	}
	if len(deps) > 3 {
		h.scim, _ = deps[3].(SCIMRepository)
	}
	if len(deps) > 0 {
		if repository, ok := deps[0].(browserauth.Repository); ok {
			h.users = repository
		}
	}
	return h
}

// Providers blocks identity provider reads and writes until Go-native identity
// cutover has passed compatibility, tenant-isolation, and rollback gates.
func (handler Handler) Providers(writer http.ResponseWriter, request *http.Request) {
	if handler.sessions != nil && handler.providers != nil {
		user, ok := browserauth.AuthenticateRequest(request.Context(), request, handler.sessions, time.Now().UTC())
		if !ok {
			httpx.WriteError(writer, request, http.StatusUnauthorized, "UNAUTHENTICATED", "An active browser session is required.")
			return
		}
		if request.Method == http.MethodGet {
			v, e := handler.providers.ListProviders(request.Context(), user.ActiveTenant())
			if e != nil {
				httpx.WriteError(writer, request, 500, "IDENTITY_UNAVAILABLE", "Identity providers could not be loaded.")
				return
			}
			httpx.WriteJSON(writer, 200, v)
			return
		}
		var in Provider
		if json.NewDecoder(http.MaxBytesReader(writer, request.Body, 64<<10)).Decode(&in) != nil || strings.TrimSpace(in.Type) == "" || strings.TrimSpace(in.Name) == "" {
			httpx.WriteError(writer, request, 400, "INVALID_IDENTITY_PROVIDER", "Provider type and name are required.")
			return
		}
		in.Type = strings.ToLower(strings.TrimSpace(in.Type))
		in.Name = strings.TrimSpace(in.Name)
		created, e := handler.providers.CreateProvider(request.Context(), user.ActiveTenant(), in)
		if e != nil {
			httpx.WriteError(writer, request, 500, "IDENTITY_UNAVAILABLE", "Identity provider could not be created.")
			return
		}
		httpx.WriteJSON(writer, 201, created)
		return
	}
	handler.writeMigrationDisabled(writer, request, "identity-providers")
}

// BreakGlass blocks break-glass account updates until the Go-native
// implementation owns browser authentication and administration.
func (handler Handler) BreakGlass(writer http.ResponseWriter, request *http.Request) {
	if !hasPathParam(request, "user") {
		httpx.WriteError(writer, request, http.StatusBadRequest, "INVALID_USER", "The user identifier is required.")
		return
	}
	handler.writeMigrationDisabled(writer, request, "break-glass")
}

// BreakGlassTest blocks break-glass verification writes until cutover.
func (handler Handler) BreakGlassTest(writer http.ResponseWriter, request *http.Request) {
	if !hasPathParam(request, "user") {
		httpx.WriteError(writer, request, http.StatusBadRequest, "INVALID_USER", "The user identifier is required.")
		return
	}
	handler.writeMigrationDisabled(writer, request, "break-glass-test")
}

// SCIMUsers blocks SCIM lifecycle writes until Go owns the identity provider.
func (handler Handler) SCIMUsers(writer http.ResponseWriter, request *http.Request) {
	if !hasPathParam(request, "identityProvider") {
		httpx.WriteError(writer, request, http.StatusBadRequest, "INVALID_IDENTITY_PROVIDER", "The identity provider identifier is required.")
		return
	}
	if !validateSignedPayload(writer, request) {
		return
	}
	if handler.scim != nil {
		var in struct {
			Name, Email string `json:"name"`
			Active      *bool  `json:"active"`
		}
		if json.NewDecoder(http.MaxBytesReader(writer, request.Body, 64<<10)).Decode(&in) != nil || !strings.Contains(in.Email, "@") {
			httpx.WriteError(writer, request, 400, "INVALID_SCIM_USER", "A valid SCIM email is required.")
			return
		}
		active := true
		if in.Active != nil {
			active = *in.Active
		}
		user, err := browserauth.AuthenticateRequest(request.Context(), request, handler.sessions, time.Now().UTC())
		if !err {
			httpx.WriteError(writer, request, 401, "UNAUTHENTICATED", "An active browser session is required.")
			return
		}
		created, e := handler.scim.CreateSCIMUser(request.Context(), user.ActiveTenant(), strings.TrimSpace(in.Email), strings.TrimSpace(in.Name), active)
		if e != nil {
			httpx.WriteError(writer, request, 500, "SCIM_UNAVAILABLE", "SCIM user could not be persisted.")
			return
		}
		httpx.WriteJSON(writer, 201, map[string]any{"id": created.ID, "userName": created.Email, "active": active})
		return
	}
	handler.writeMigrationDisabled(writer, request, "scim-users")
}

// MFAEnroll blocks MFA enrollment until Go-native browser authentication is enabled.
func (handler Handler) MFAEnroll(writer http.ResponseWriter, request *http.Request) {
	if handler.mfa != nil && handler.sessions != nil {
		user, ok := browserauth.AuthenticateRequest(request.Context(), request, handler.sessions, time.Now().UTC())
		if !ok {
			httpx.WriteError(writer, request, 401, "UNAUTHENTICATED", "An active browser session is required.")
			return
		}
		secret, err := newTOTPSecret()
		if err != nil || handler.mfa.SetMFASecret(request.Context(), user.ID, encryptSecret(secret)) != nil {
			httpx.WriteError(writer, request, 500, "MFA_UNAVAILABLE", "MFA enrollment could not be started.")
			return
		}
		httpx.WriteJSON(writer, 201, map[string]string{"secret": secret, "algorithm": "SHA1", "digits": "6", "period": "30"})
		return
	}
	handler.writeMigrationDisabled(writer, request, "mfa-enroll")
}

// MFAConfirm blocks MFA confirmation until Go-native browser authentication is enabled.
func (handler Handler) MFAConfirm(writer http.ResponseWriter, request *http.Request) {
	if handler.verifyMFA(writer, request, true) {
		return
	}
	handler.writeMigrationDisabled(writer, request, "mfa-confirm")
}

// MFAStepUp blocks MFA step-up until Go-native browser authentication is enabled.
func (handler Handler) MFAStepUp(writer http.ResponseWriter, request *http.Request) {
	if handler.verifyMFA(writer, request, false) {
		return
	}
	handler.writeMigrationDisabled(writer, request, "mfa-step-up")
}

func (handler Handler) verifyMFA(w http.ResponseWriter, r *http.Request, confirm bool) bool {
	if handler.mfa == nil || handler.sessions == nil {
		return false
	}
	u, ok := browserauth.AuthenticateRequest(r.Context(), r, handler.sessions, time.Now().UTC())
	if !ok {
		httpx.WriteError(w, r, 401, "UNAUTHENTICATED", "An active browser session is required.")
		return true
	}
	var in struct {
		Code string `json:"code"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil || len(in.Code) != 6 {
		httpx.WriteError(w, r, 400, "INVALID_MFA_CODE", "A six-digit MFA code is required.")
		return true
	}
	enc, e := handler.mfa.GetMFASecret(r.Context(), u.ID)
	if e != nil {
		httpx.WriteError(w, r, 400, "MFA_NOT_ENROLLED", "MFA enrollment is required.")
		return true
	}
	secret, e := decryptSecret(enc)
	if e != nil || !validTOTP(secret, in.Code, time.Now().UTC()) {
		httpx.WriteError(w, r, 401, "INVALID_MFA_CODE", "The MFA code is invalid.")
		return true
	}
	if confirm {
		if e = handler.mfa.MarkMFAConfirmed(r.Context(), u.ID, time.Now().UTC()); e != nil {
			httpx.WriteError(w, r, 500, "MFA_UNAVAILABLE", "MFA confirmation could not be saved.")
			return true
		}
	}
	httpx.WriteJSON(w, 200, map[string]any{"verified": true, "stepUp": !confirm})
	return true
}
func newTOTPSecret() (string, error) {
	b := make([]byte, 20)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}
func encryptionKey() []byte {
	raw := os.Getenv("IDELIUM_MFA_ENCRYPTION_KEY")
	b, _ := hex.DecodeString(raw)
	if len(b) == 32 {
		return b
	}
	return nil
}
func encryptSecret(secret string) string {
	key := encryptionKey()
	if len(key) != 32 {
		return ""
	}
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	_, _ = rand.Read(nonce)
	out := gcm.Seal(nonce, nonce, []byte(secret), nil)
	return hex.EncodeToString(out)
}
func decryptSecret(value string) (string, error) {
	key := encryptionKey()
	if len(key) != 32 {
		return "", fmt.Errorf("MFA encryption key unavailable")
	}
	raw, e := hex.DecodeString(value)
	if e != nil {
		return "", e
	}
	block, e := aes.NewCipher(key)
	if e != nil {
		return "", e
	}
	gcm, e := cipher.NewGCM(block)
	if e != nil {
		return "", e
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("invalid encrypted MFA secret")
	}
	plain, e := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	return string(plain), e
}
func validTOTP(secret, code string, now time.Time) bool {
	key, e := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if e != nil || len(code) != 6 {
		return false
	}
	for offset := -1; offset <= 1; offset++ {
		counter := uint64(now.Unix()/30 + int64(offset))
		buf := make([]byte, 8)
		for i := 7; i >= 0; i-- {
			buf[i] = byte(counter)
			counter >>= 8
		}
		mac := hmac.New(sha1.New, key)
		_, _ = mac.Write(buf)
		sum := mac.Sum(nil)
		n := int(sum[len(sum)-1] & 15)
		value := (int(sum[n])<<24 | int(sum[n+1])<<16 | int(sum[n+2])<<8 | int(sum[n+3])) & 0x7fffffff
		if fmt.Sprintf("%06d", value%1000000) == code {
			return true
		}
	}
	return false
}

// OIDCTokenExchange blocks workload identity exchange until Go-native trust
// validation is enabled.
func (handler Handler) OIDCTokenExchange(writer http.ResponseWriter, request *http.Request) {
	if !validateSignedPayload(writer, request) {
		return
	}
	handler.writeMigrationDisabled(writer, request, "oidc-token-exchange")
}

// SSOStart blocks SSO bootstrap until Go-native SSO is enabled.
func (handler Handler) SSOStart(writer http.ResponseWriter, request *http.Request) {
	if !hasPathParam(request, "identityProvider") {
		httpx.WriteError(writer, request, http.StatusBadRequest, "INVALID_IDENTITY_PROVIDER", "The identity provider identifier is required.")
		return
	}
	handler.writeMigrationDisabled(writer, request, "sso-start")
}

// OIDCCallback blocks OIDC callbacks until Go-native SSO is enabled.
func (handler Handler) OIDCCallback(writer http.ResponseWriter, request *http.Request) {
	if !hasPathParam(request, "identityProvider") {
		httpx.WriteError(writer, request, http.StatusBadRequest, "INVALID_IDENTITY_PROVIDER", "The identity provider identifier is required.")
		return
	}
	if !validateSignedPayload(writer, request) {
		return
	}
	if handler.users != nil {
		handler.issueSSOSession(writer, request)
		return
	}
	handler.writeMigrationDisabled(writer, request, "oidc-callback")
}

// SAMLCallback blocks SAML callbacks until Go-native SSO is enabled.
func (handler Handler) SAMLCallback(writer http.ResponseWriter, request *http.Request) {
	if !hasPathParam(request, "identityProvider") {
		httpx.WriteError(writer, request, http.StatusBadRequest, "INVALID_IDENTITY_PROVIDER", "The identity provider identifier is required.")
		return
	}
	if !validateSignedPayload(writer, request) {
		return
	}
	if handler.users != nil {
		handler.issueSSOSession(writer, request)
		return
	}
	handler.writeMigrationDisabled(writer, request, "saml-callback")
}

func (handler Handler) issueSSOSession(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&in) != nil || !strings.Contains(in.Email, "@") {
		httpx.WriteError(w, r, 400, "INVALID_SSO_ASSERTION", "A validated subject email is required.")
		return
	}
	user, e := handler.users.FindByEmail(r.Context(), strings.TrimSpace(in.Email))
	if e != nil || user.Status != "active" {
		httpx.WriteError(w, r, 401, "SSO_USER_NOT_FOUND", "The validated SSO subject is not an active user.")
		return
	}
	sid := make([]byte, 32)
	csrf := make([]byte, 32)
	if _, e = rand.Read(sid); e != nil {
		httpx.WriteError(w, r, 500, "SSO_UNAVAILABLE", "SSO session could not be created.")
		return
	}
	if _, e = rand.Read(csrf); e != nil {
		httpx.WriteError(w, r, 500, "SSO_UNAVAILABLE", "SSO session could not be created.")
		return
	}
	session := browserauth.Session{ID: hex.EncodeToString(sid), UserID: user.ID, TenantID: user.TenantID, CSRFToken: hex.EncodeToString(csrf), ExpiresAt: time.Now().UTC().Add(8 * time.Hour)}
	if e = handler.sessions.Create(r.Context(), session); e != nil {
		httpx.WriteError(w, r, 500, "SSO_UNAVAILABLE", "SSO session could not be created.")
		return
	}
	browserauth.SetSessionCookies(w, session.ID, session.CSRFToken)
	httpx.WriteJSON(w, 200, map[string]any{"authenticated": true, "userId": user.ID})
}

// validateSignedPayload verifies an HMAC signature and single-use nonce before
// any identity assertion is considered. Payload bytes are never logged or echoed.
func validateSignedPayload(writer http.ResponseWriter, request *http.Request) bool {
	secret := os.Getenv("IDELIUM_IDENTITY_CALLBACK_SECRET")
	nonce := strings.TrimSpace(request.Header.Get("Idelium-Identity-Nonce"))
	signature := strings.TrimSpace(request.Header.Get("Idelium-Identity-Signature"))
	body, err := io.ReadAll(http.MaxBytesReader(writer, request.Body, 256<<10))
	request.Body = io.NopCloser(bytes.NewReader(body))
	if secret == "" || nonce == "" || signature == "" || err != nil {
		httpx.WriteError(writer, request, http.StatusUnauthorized, "IDENTITY_SIGNATURE_REQUIRED", "A signed identity payload is required.")
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	if !hmac.Equal([]byte(signature), []byte(fmt.Sprintf("%x", mac.Sum(nil)))) {
		httpx.WriteError(writer, request, http.StatusUnauthorized, "IDENTITY_SIGNATURE_INVALID", "The identity payload signature is invalid.")
		return false
	}
	replayMu.Lock()
	defer replayMu.Unlock()
	now := time.Now().UTC()
	for key, at := range replayedNonces {
		if now.Sub(at) > 10*time.Minute {
			delete(replayedNonces, key)
		}
	}
	if _, exists := replayedNonces[nonce]; exists {
		httpx.WriteError(writer, request, http.StatusConflict, "IDENTITY_REPLAY_DETECTED", "The identity payload nonce was already used.")
		return false
	}
	if len(replayedNonces) >= 10000 {
		httpx.WriteError(writer, request, http.StatusServiceUnavailable, "IDENTITY_REPLAY_CACHE_FULL", "Identity replay protection is temporarily unavailable.")
		return false
	}
	replayedNonces[nonce] = now
	return true
}

func (handler Handler) writeMigrationDisabled(
	writer http.ResponseWriter,
	request *http.Request,
	surface string,
) {
	if handler.logger != nil {
		handler.logger.Info(
			"Advanced identity route rejected before Go-native cutover",
			"surface", surface,
			"correlation_id", httpx.GetCorrelationID(request.Context()),
		)
	}
	httpx.WriteError(
		writer,
		request,
		http.StatusConflict,
		"IDENTITY_LARAVEL_OWNER",
		"Advanced identity operations remain owned by the Laravel runtime.",
	)
}

func hasPathParam(request *http.Request, name string) bool {
	return chi.URLParam(request, name) != ""
}
