package artifacts_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/idelium/idelium-api-go/internal/artifacts"
)

func TestValidateDescriptorAcceptsLaravelCompatibleArtifact(t *testing.T) {
	policy := artifacts.DefaultPolicy()

	violations := policy.ValidateDescriptor(artifacts.Descriptor{
		ArtifactType:   "postman-report",
		Name:           "newman-report.json",
		ContentType:    "application/json",
		SizeBytes:      artifacts.DefaultMaxArtifactBytes,
		ChecksumSHA256: strings.Repeat("a", 64),
		StorageKey:     "tenant/42/run/7/newman-report.json",
	})

	if len(violations) != 0 {
		t.Fatalf("expected descriptor to pass, got %#v", violations)
	}
}

func TestValidateDescriptorRejectsOversizedArtifactSafely(t *testing.T) {
	policy := artifacts.DefaultPolicy()
	unsafeStorageKey := "tenant/secret-customer/run/7/screenshot.png"
	unsafeChecksum := strings.Repeat("b", 64)

	violations := policy.ValidateDescriptor(artifacts.Descriptor{
		ArtifactType:   "screenshot",
		Name:           "screenshot.png",
		ContentType:    "image/png",
		SizeBytes:      artifacts.DefaultMaxArtifactBytes + 1,
		ChecksumSHA256: unsafeChecksum,
		StorageKey:     unsafeStorageKey,
	})

	assertViolationCode(t, violations, "ARTIFACT_SIZE_LIMIT_EXCEEDED")
	payload, err := json.Marshal(violations)
	if err != nil {
		t.Fatalf("marshal violations: %v", err)
	}
	diagnostic := string(payload)
	if strings.Contains(diagnostic, unsafeStorageKey) || strings.Contains(diagnostic, unsafeChecksum) ||
		strings.Contains(diagnostic, "secret-customer") {
		t.Fatalf("artifact policy leaked unsafe descriptor values: %s", diagnostic)
	}
}

func TestValidateDescriptorRejectsUnsupportedContentTypeAndChecksum(t *testing.T) {
	violations := artifacts.DefaultPolicy().ValidateDescriptor(artifacts.Descriptor{
		ArtifactType:   "binary-dump",
		Name:           "dump.bin",
		ContentType:    "application/octet-stream",
		SizeBytes:      1024,
		ChecksumSHA256: "not-a-checksum",
		StorageKey:     "tenant/42/run/7/dump.bin",
	})

	assertViolationCode(t, violations, "ARTIFACT_CONTENT_TYPE_NOT_ALLOWED")
	assertViolationCode(t, violations, "ARTIFACT_CHECKSUM_INVALID")
}

func TestValidateInlineArtifactAndCollectionLimits(t *testing.T) {
	policy := artifacts.DefaultPolicy()

	inlineViolations := policy.ValidateInlineArtifact("artifacts[0].data", artifacts.DefaultInlineArtifactMaxBytes+1)
	assertViolationCode(t, inlineViolations, "ARTIFACT_INLINE_SIZE_LIMIT_EXCEEDED")

	collectionViolations := policy.ValidateCollection("artifacts", artifacts.DefaultCollectionMaxItems+1)
	assertViolationCode(t, collectionViolations, "ARTIFACT_COLLECTION_LIMIT_EXCEEDED")
}

func TestRetentionDurationUsesDefaultRetentionWindow(t *testing.T) {
	duration := artifacts.DefaultPolicy().RetentionDuration()

	if duration.Hours() != 24*artifacts.DefaultRetentionDays {
		t.Fatalf("unexpected retention duration: %s", duration)
	}
}

func assertViolationCode(t *testing.T, violations []artifacts.Violation, code string) {
	t.Helper()

	for _, violation := range violations {
		if violation.Code == code {
			return
		}
	}
	t.Fatalf("expected violation code %s, got %#v", code, violations)
}
