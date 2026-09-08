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
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
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
	logger     *slog.Logger
	sessions   browserauth.SessionRepository
	providers  ProviderRepository
	mfa        MFARepository
	scim       SCIMRepository
	users      browserauth.UserRepository
	sso        SSOStateRepository
	service    ServiceAccountBinding
	breakglass BreakGlassRepository
}

type Provider struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Issuer   string `json:"issuer,omitempty"`
	Audience string `json:"audience,omitempty"`
	Status   string `json:"status"`
}

var ErrProviderNotFound = errors.New("identity provider not found")

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
	UpdateSCIMUser(context.Context, int64, int64, string, string, bool) (browserauth.User, error)
	DeleteSCIMUser(context.Context, int64, int64) error
}
type SSOStateRepository interface {
	CreateSSOState(context.Context, int64, int64, string, string, time.Time) error
	ConsumeSSOState(context.Context, string, time.Time) (int64, int64, error)
	Provider(context.Context, int64, string) (Provider, error)
}

// SSOStateInspector exposes the unconsumed state binding so token claims can
// be checked against the tenant and provider before the state is spent.
type SSOStateInspector interface {
	SSOState(context.Context, string, time.Time) (int64, int64, error)
}

type ServiceAccountBinding interface {
	ActiveServiceAccount(context.Context, int64, string, time.Time) (bool, error)
}

type BreakGlassRepository interface {
	SetBreakGlass(context.Context, int64, int64, string, time.Time, time.Time) error
	TestBreakGlass(context.Context, int64, int64, time.Time) (bool, error)
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
	if len(deps) > 4 {
		h.sso, _ = deps[4].(SSOStateRepository)
	}
	if len(deps) > 5 {
		h.service, _ = deps[5].(ServiceAccountBinding)
	}
	if len(deps) > 6 {
		h.breakglass, _ = deps[6].(BreakGlassRepository)
	}
	if len(deps) > 7 {
		h.breakglass, _ = deps[7].(BreakGlassRepository)
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
	if handler.breakglass == nil || handler.sessions == nil {
		httpx.WriteError(writer, request, 503, "BREAK_GLASS_UNAVAILABLE", "Break-glass storage is unavailable.")
		return
	}
	actor, ok := browserauth.AuthenticateRequest(request.Context(), request, handler.sessions, time.Now().UTC())
	if !ok {
		httpx.WriteError(writer, request, 401, "UNAUTHENTICATED", "An active browser session is required.")
		return
	}
	target, err := strconv.ParseInt(chi.URLParam(request, "user"), 10, 64)
	if err != nil || target <= 0 {
		httpx.WriteError(writer, request, 400, "INVALID_USER", "The user identifier is required.")
		return
	}
	var in struct {
		Reason    string    `json:"reason"`
		ExpiresAt time.Time `json:"expiresAt"`
	}
	if json.NewDecoder(http.MaxBytesReader(writer, request.Body, 16<<10)).Decode(&in) != nil || strings.TrimSpace(in.Reason) == "" || in.ExpiresAt.IsZero() || !in.ExpiresAt.After(time.Now().UTC()) || in.ExpiresAt.After(time.Now().UTC().Add(time.Hour)) {
		httpx.WriteError(writer, request, 422, "INVALID_BREAK_GLASS", "Reason and an expiry within one hour are required.")
		return
	}
	if err = handler.breakglass.SetBreakGlass(request.Context(), actor.ActiveTenant(), target, strings.TrimSpace(in.Reason), in.ExpiresAt.UTC(), time.Now().UTC()); err != nil {
		httpx.WriteError(writer, request, 404, "USER_NOT_FOUND", "User not found.")
		return
	}
	httpx.WriteJSON(writer, 200, map[string]any{"user": target, "expiresAt": in.ExpiresAt.UTC(), "enabled": true})
}

// BreakGlassTest blocks break-glass verification writes until cutover.
func (handler Handler) BreakGlassTest(writer http.ResponseWriter, request *http.Request) {
	if !hasPathParam(request, "user") {
		httpx.WriteError(writer, request, http.StatusBadRequest, "INVALID_USER", "The user identifier is required.")
		return
	}
	if handler.breakglass == nil || handler.sessions == nil {
		httpx.WriteError(writer, request, 503, "BREAK_GLASS_UNAVAILABLE", "Break-glass storage is unavailable.")
		return
	}
	actor, ok := browserauth.AuthenticateRequest(request.Context(), request, handler.sessions, time.Now().UTC())
	if !ok {
		httpx.WriteError(writer, request, 401, "UNAUTHENTICATED", "An active browser session is required.")
		return
	}
	target, err := strconv.ParseInt(chi.URLParam(request, "user"), 10, 64)
	if err != nil || target <= 0 {
		httpx.WriteError(writer, request, 400, "INVALID_USER", "The user identifier is required.")
		return
	}
	valid, err := handler.breakglass.TestBreakGlass(request.Context(), actor.ActiveTenant(), target, time.Now().UTC())
	if err != nil {
		httpx.WriteError(writer, request, 404, "USER_NOT_FOUND", "User not found.")
		return
	}
	if !valid {
		httpx.WriteError(writer, request, 409, "BREAK_GLASS_INACTIVE", "Break-glass control is inactive or expired.")
		return
	}
	httpx.WriteJSON(writer, 200, map[string]any{"user": target, "valid": true})
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
		user, err := browserauth.AuthenticateRequest(request.Context(), request, handler.sessions, time.Now().UTC())
		if !err {
			httpx.WriteError(writer, request, 401, "UNAUTHENTICATED", "An active browser session is required.")
			return
		}
		providerID, parseProviderErr := strconv.ParseInt(chi.URLParam(request, "identityProvider"), 10, 64)
		if parseProviderErr != nil || providerID <= 0 || handler.providers == nil {
			httpx.WriteError(writer, request, 404, "IDENTITY_PROVIDER_NOT_FOUND", "Identity provider not found.")
			return
		}
		providers, listErr := handler.providers.ListProviders(request.Context(), user.ActiveTenant())
		providerOK := false
		for _, provider := range providers {
			if provider.ID == providerID && provider.Status == "active" {
				providerOK = true
				break
			}
		}
		if listErr != nil || !providerOK {
			httpx.WriteError(writer, request, 404, "IDENTITY_PROVIDER_NOT_FOUND", "Identity provider not found.")
			return
		}
		userID, _ := strconv.ParseInt(chi.URLParam(request, "user"), 10, 64)
		if request.Method == http.MethodDelete {
			if userID <= 0 {
				httpx.WriteError(writer, request, 400, "INVALID_SCIM_USER", "The SCIM user identifier is required.")
				return
			}
			if e := handler.scim.DeleteSCIMUser(request.Context(), user.ActiveTenant(), userID); e != nil {
				httpx.WriteError(writer, request, 404, "SCIM_USER_NOT_FOUND", "SCIM user not found.")
				return
			}
			httpx.WriteJSON(writer, 200, map[string]any{"id": userID, "deleted": true})
			return
		}
		var in struct {
			ID          int64  `json:"id"`
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
		if request.Method == http.MethodPut || request.Method == http.MethodPatch {
			if userID <= 0 {
				userID = in.ID
			}
			if userID <= 0 {
				httpx.WriteError(writer, request, 400, "INVALID_SCIM_USER", "The SCIM user identifier is required.")
				return
			}
			updated, e := handler.scim.UpdateSCIMUser(request.Context(), user.ActiveTenant(), userID, strings.TrimSpace(in.Email), strings.TrimSpace(in.Name), active)
			if e != nil {
				httpx.WriteError(writer, request, 404, "SCIM_USER_NOT_FOUND", "SCIM user not found.")
				return
			}
			httpx.WriteJSON(writer, 200, map[string]any{"id": updated.ID, "userName": updated.Email, "active": active})
			return
		}
		if in.ID > 0 {
			updated, e := handler.scim.UpdateSCIMUser(request.Context(), user.ActiveTenant(), in.ID, strings.TrimSpace(in.Email), strings.TrimSpace(in.Name), active)
			if e != nil {
				httpx.WriteError(writer, request, 404, "SCIM_USER_NOT_FOUND", "SCIM user not found.")
				return
			}
			httpx.WriteJSON(writer, 200, map[string]any{"id": updated.ID, "userName": updated.Email, "active": active})
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

// OIDCTokenExchange validates a signed OIDC workload token and consumes its
// persisted SSO state exactly once.
func (handler Handler) OIDCTokenExchange(writer http.ResponseWriter, request *http.Request) {
	if !validateSignedPayload(writer, request) {
		return
	}
	var in struct {
		IDToken string `json:"id_token"`
		State   string `json:"state"`
		Nonce   string `json:"nonce"`
	}
	if json.NewDecoder(request.Body).Decode(&in) != nil || in.IDToken == "" || in.State == "" || in.Nonce == "" {
		httpx.WriteError(writer, request, 400, "INVALID_OIDC_REQUEST", "id_token, state and nonce are required.")
		return
	}
	if handler.sso == nil {
		httpx.WriteError(writer, request, 503, "OIDC_UNAVAILABLE", "OIDC state storage is unavailable.")
		return
	}
	var tenantID, providerID int64
	var err error
	if inspector, ok := handler.sso.(SSOStateInspector); ok {
		tenantID, providerID, err = inspector.SSOState(request.Context(), in.State, time.Now().UTC())
		if err != nil {
			httpx.WriteError(writer, request, 401, "INVALID_OIDC_STATE", "The OIDC state is invalid or expired.")
			return
		}
		provider, providerErr := handler.sso.Provider(request.Context(), tenantID, strconv.FormatInt(providerID, 10))
		if providerErr != nil || provider.Status != "active" || provider.Issuer == "" || provider.Audience == "" {
			httpx.WriteError(writer, request, 401, "INVALID_OIDC_PROVIDER", "The OIDC provider binding is invalid.")
			return
		}
		claims, err := validateOIDCJWTForProvider(in.IDToken, in.Nonce, provider.Issuer, provider.Audience)
		if err != nil {
			httpx.WriteError(writer, request, 401, "INVALID_OIDC_TOKEN", err.Error())
			return
		}
		if handler.service == nil {
			httpx.WriteError(writer, request, 503, "OIDC_UNAVAILABLE", "Service-account binding is unavailable.")
			return
		}
		bound, bindingErr := handler.service.ActiveServiceAccount(request.Context(), tenantID, claims.Subject, time.Now().UTC())
		if bindingErr != nil || !bound {
			httpx.WriteError(writer, request, 401, "INVALID_SERVICE_ACCOUNT", "The OIDC subject is not bound to an active service account.")
			return
		}
		if _, _, err = handler.sso.ConsumeSSOState(request.Context(), in.State, time.Now().UTC()); err != nil {
			httpx.WriteError(writer, request, 401, "INVALID_OIDC_STATE", "The OIDC state is invalid or expired.")
			return
		}
		httpx.WriteJSON(writer, 200, map[string]any{"validated": true, "tenantId": tenantID, "providerId": providerID, "issuer": claims.Issuer, "subject": claims.Subject, "email": claims.Email})
		return
	}
	claims, err := validateOIDCJWT(in.IDToken, in.Nonce)
	if err != nil {
		httpx.WriteError(writer, request, 401, "INVALID_OIDC_TOKEN", err.Error())
		return
	}
	if _, _, err = handler.sso.ConsumeSSOState(request.Context(), in.State, time.Now().UTC()); err != nil {
		httpx.WriteError(writer, request, 401, "INVALID_OIDC_STATE", "The OIDC state is invalid or expired.")
		return
	}
	httpx.WriteJSON(writer, 200, map[string]any{"validated": true, "issuer": claims.Issuer, "subject": claims.Subject, "email": claims.Email})
	return
}

type oidcClaims struct {
	Issuer   string `json:"iss"`
	Subject  string `json:"sub"`
	Audience any    `json:"aud"`
	Exp      int64  `json:"exp"`
	Nonce    string `json:"nonce"`
	Email    string `json:"email"`
}

func validateOIDCJWT(token, expectedNonce string) (oidcClaims, error) {
	return validateOIDCJWTForProvider(token, expectedNonce, os.Getenv("IDELIUM_OIDC_ISSUER"), os.Getenv("IDELIUM_OIDC_AUDIENCE"))
}

func validateOIDCJWTForProvider(token, expectedNonce, expectedIssuer, expectedAudience string) (oidcClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return oidcClaims{}, fmt.Errorf("malformed JWT")
	}
	decode := func(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) }
	head, e := decode(parts[0])
	if e != nil {
		return oidcClaims{}, fmt.Errorf("malformed JWT header")
	}
	var h struct {
		Alg string `json:"alg"`
	}
	if json.Unmarshal(head, &h) != nil || h.Alg != "HS256" {
		return oidcClaims{}, fmt.Errorf("unsupported JWT algorithm")
	}
	payload, e := decode(parts[1])
	if e != nil {
		return oidcClaims{}, fmt.Errorf("malformed JWT claims")
	}
	var c oidcClaims
	if json.Unmarshal(payload, &c) != nil || c.Issuer == "" || c.Subject == "" || c.Exp <= time.Now().Unix() || c.Nonce != expectedNonce {
		return oidcClaims{}, fmt.Errorf("invalid JWT claims")
	}
	expectedIssuer = strings.TrimSpace(expectedIssuer)
	if expectedIssuer == "" || c.Issuer != expectedIssuer {
		return oidcClaims{}, fmt.Errorf("invalid JWT issuer")
	}
	expectedAudience = strings.TrimSpace(expectedAudience)
	if expectedAudience == "" {
		return oidcClaims{}, fmt.Errorf("OIDC audience is not configured")
	}
	audOK := false
	switch a := c.Audience.(type) {
	case string:
		audOK = a == expectedAudience
	case []any:
		for _, v := range a {
			if s, _ := v.(string); s == expectedAudience {
				audOK = true
			}
		}
	}
	if !audOK {
		return oidcClaims{}, fmt.Errorf("invalid JWT audience")
	}
	secret := os.Getenv("IDELIUM_OIDC_HS256_SECRET")
	if secret == "" {
		return oidcClaims{}, fmt.Errorf("OIDC verification key unavailable")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	sig, e := decode(parts[2])
	if e != nil || !hmac.Equal(sig, mac.Sum(nil)) {
		return oidcClaims{}, fmt.Errorf("invalid JWT signature")
	}
	return c, nil
}

// SSOStart blocks SSO bootstrap until Go-native SSO is enabled.
func (handler Handler) SSOStart(writer http.ResponseWriter, request *http.Request) {
	if !hasPathParam(request, "identityProvider") {
		httpx.WriteError(writer, request, http.StatusBadRequest, "INVALID_IDENTITY_PROVIDER", "The identity provider identifier is required.")
		return
	}
	if handler.sso == nil || handler.sessions == nil {
		handler.writeMigrationDisabled(writer, request, "sso-start")
		return
	}
	user, ok := browserauth.AuthenticateRequest(request.Context(), request, handler.sessions, time.Now().UTC())
	if !ok {
		httpx.WriteError(writer, request, 401, "UNAUTHENTICATED", "An active browser session is required.")
		return
	}
	provider, err := handler.sso.Provider(request.Context(), user.ActiveTenant(), chi.URLParam(request, "identityProvider"))
	if err != nil || provider.Status != "active" {
		httpx.WriteError(writer, request, 404, "IDENTITY_PROVIDER_NOT_FOUND", "Identity provider not found.")
		return
	}
	stateBytes := make([]byte, 32)
	challengeBytes := make([]byte, 32)
	if _, err = rand.Read(stateBytes); err != nil {
		httpx.WriteError(writer, request, 500, "SSO_UNAVAILABLE", "SSO state could not be created.")
		return
	}
	if _, err = rand.Read(challengeBytes); err != nil {
		httpx.WriteError(writer, request, 500, "SSO_UNAVAILABLE", "SSO state could not be created.")
		return
	}
	state := hex.EncodeToString(stateBytes)
	challenge := hex.EncodeToString(challengeBytes)
	expires := time.Now().UTC().Add(10 * time.Minute)
	if err = handler.sso.CreateSSOState(request.Context(), user.ActiveTenant(), provider.ID, state, challenge, expires); err != nil {
		httpx.WriteError(writer, request, 500, "SSO_UNAVAILABLE", "SSO state could not be persisted.")
		return
	}
	httpx.WriteJSON(writer, 200, map[string]any{"state": state, "codeChallenge": challenge, "expiresAt": expires, "provider": provider.Name})
	return
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
	if !handler.validateCallbackBinding(writer, request) {
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
	if !handler.validateCallbackBinding(writer, request) {
		return
	}
	if handler.users != nil {
		handler.issueSSOSession(writer, request)
		return
	}
	handler.writeMigrationDisabled(writer, request, "saml-callback")
}

func (handler Handler) validateCallbackBinding(writer http.ResponseWriter, request *http.Request) bool {
	if handler.sso == nil {
		httpx.WriteError(writer, request, 503, "SSO_UNAVAILABLE", "SSO state storage is unavailable.")
		return false
	}
	inspector, ok := handler.sso.(SSOStateInspector)
	if !ok {
		httpx.WriteError(writer, request, 503, "SSO_UNAVAILABLE", "SSO state inspection is unavailable.")
		return false
	}
	body, err := io.ReadAll(http.MaxBytesReader(writer, request.Body, 64<<10))
	request.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		httpx.WriteError(writer, request, 400, "INVALID_SSO_ASSERTION", "The SSO assertion is invalid.")
		return false
	}
	var in struct {
		State string `json:"state"`
	}
	if json.Unmarshal(body, &in) != nil || strings.TrimSpace(in.State) == "" {
		httpx.WriteError(writer, request, 400, "INVALID_SSO_ASSERTION", "A valid SSO state is required.")
		return false
	}
	tenant, provider, err := inspector.SSOState(request.Context(), in.State, time.Now().UTC())
	if err != nil {
		httpx.WriteError(writer, request, 401, "INVALID_OIDC_STATE", "The SSO state is invalid or expired.")
		return false
	}
	configured, err := handler.sso.Provider(request.Context(), tenant, chi.URLParam(request, "identityProvider"))
	if err != nil || configured.ID != provider || configured.Status != "active" {
		httpx.WriteError(writer, request, 401, "INVALID_OIDC_PROVIDER", "The SSO provider binding is invalid.")
		return false
	}
	if _, _, err = handler.sso.ConsumeSSOState(request.Context(), in.State, time.Now().UTC()); err != nil {
		httpx.WriteError(writer, request, 401, "INVALID_OIDC_STATE", "The SSO state is invalid or expired.")
		return false
	}
	return true
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
