package mysql

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSafeDatabaseFailureRedactsDriverDetails(t *testing.T) {
	secret := "database-password-must-not-appear"
	err := safeDatabaseFailure("database startup", errors.New("dsn password="+secret))

	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "dsn") {
		t.Fatalf("database diagnostic exposed driver details: %v", err)
	}
	if !strings.Contains(err.Error(), "database unavailable") {
		t.Fatalf("database diagnostic lost its safe classification: %v", err)
	}
}

func TestSafeDatabaseFailureClassifiesDeadline(t *testing.T) {
	err := safeDatabaseFailure("database startup", context.DeadlineExceeded)
	if !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("deadline diagnostic was not preserved safely: %v", err)
	}
}
