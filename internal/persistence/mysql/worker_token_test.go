package mysql

import (
	"net/http"
	"testing"
	"time"

	"github.com/idelium/idelium-api-go/internal/browserauth"
)

func TestValidateWorkerTokenRejectsMissingToken(t *testing.T) {
	repository := &BrowserAuthRepository{}
	request, err := http.NewRequest(http.MethodPost, "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.ValidateWorkerToken(request, 9001, 9001, 9001, "fixture-agent-smoke", "", time.Now().UTC()); err != browserauth.ErrWorkerTokenInvalid {
		t.Fatalf("expected missing worker token rejection, got %v", err)
	}
}
