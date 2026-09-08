package identity

import (
	"testing"
	"time"
)

func TestValidTOTPAcceptsRFCSecretWithinWindow(t *testing.T) {
	if !validTOTP("JBSWY3DPEHPK3PXP", "324550", time.Unix(1700000000, 0)) {
		t.Fatal("expected valid TOTP")
	}
	if validTOTP("JBSWY3DPEHPK3PXP", "000000", time.Unix(1700000000, 0)) {
		t.Fatal("unexpected valid TOTP")
	}
}
