// Package artifacts defines storage and size policy for execution artifacts.
package artifacts

import (
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	// DefaultMaxArtifactBytes matches the Laravel artifact descriptor limit.
	DefaultMaxArtifactBytes int64 = 50 * 1024 * 1024
	// DefaultInlineArtifactMaxBytes matches the Laravel result payload inline limit.
	DefaultInlineArtifactMaxBytes int64 = 262144
	// DefaultCollectionMaxItems matches the Laravel result payload artifact count limit.
	DefaultCollectionMaxItems = 50
	// DefaultRetentionDays matches the Laravel artifact retention default.
	DefaultRetentionDays = 30
)

var checksumPattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

// Policy contains the bounded artifact limits applied before storing artifact
// descriptors or accepting inline result artifacts.
type Policy struct {
	MaxArtifactBytes       int64
	InlineArtifactMaxBytes int64
	CollectionMaxItems     int
	DefaultRetentionDays   int
	AllowedContentTypes    []string
}

// Descriptor describes artifact metadata before storage.
type Descriptor struct {
	ArtifactType   string
	Name           string
	ContentType    string
	SizeBytes      int64
	ChecksumSHA256 string
	StorageKey     string
}

// Violation is a safe diagnostic that never includes artifact payloads, storage
// keys, checksums, or tenant identifiers.
type Violation struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Limit   int64  `json:"limit,omitempty"`
	Actual  int64  `json:"actual,omitempty"`
}

// DefaultPolicy returns the Laravel-compatible artifact policy used during the
// migration coexistence window.
func DefaultPolicy() Policy {
	return Policy{
		MaxArtifactBytes:       DefaultMaxArtifactBytes,
		InlineArtifactMaxBytes: DefaultInlineArtifactMaxBytes,
		CollectionMaxItems:     DefaultCollectionMaxItems,
		DefaultRetentionDays:   DefaultRetentionDays,
		AllowedContentTypes: []string{
			"application/json",
			"application/junit+xml",
			"application/xml",
			"text/markdown",
			"text/html",
			"text/plain",
			"image/png",
			"image/jpeg",
		},
	}
}

// ValidateDescriptor checks descriptor metadata before it is persisted.
func (policy Policy) ValidateDescriptor(descriptor Descriptor) []Violation {
	policy = policy.normalized()
	violations := make([]Violation, 0)

	if strings.TrimSpace(descriptor.ArtifactType) == "" {
		violations = append(violations, required("artifactType"))
	}
	if strings.TrimSpace(descriptor.Name) == "" {
		violations = append(violations, required("name"))
	}
	if strings.TrimSpace(descriptor.ContentType) == "" {
		violations = append(violations, required("contentType"))
	} else if !slices.Contains(policy.AllowedContentTypes, descriptor.ContentType) {
		violations = append(violations, Violation{
			Field:   "contentType",
			Code:    "ARTIFACT_CONTENT_TYPE_NOT_ALLOWED",
			Message: "The artifact content type is not allowed.",
		})
	}
	if descriptor.SizeBytes < 0 {
		violations = append(violations, Violation{
			Field:   "sizeBytes",
			Code:    "ARTIFACT_SIZE_INVALID",
			Message: "The artifact size must not be negative.",
			Actual:  descriptor.SizeBytes,
		})
	} else if descriptor.SizeBytes > policy.MaxArtifactBytes {
		violations = append(violations, Violation{
			Field:   "sizeBytes",
			Code:    "ARTIFACT_SIZE_LIMIT_EXCEEDED",
			Message: "The artifact exceeds the configured size limit.",
			Limit:   policy.MaxArtifactBytes,
			Actual:  descriptor.SizeBytes,
		})
	}
	if strings.TrimSpace(descriptor.ChecksumSHA256) == "" {
		violations = append(violations, required("checksumSha256"))
	} else if !checksumPattern.MatchString(descriptor.ChecksumSHA256) {
		violations = append(violations, Violation{
			Field:   "checksumSha256",
			Code:    "ARTIFACT_CHECKSUM_INVALID",
			Message: "The artifact checksum must be a SHA-256 hex digest.",
		})
	}
	if strings.TrimSpace(descriptor.StorageKey) == "" {
		violations = append(violations, required("storageKey"))
	}

	return violations
}

// ValidateInlineArtifact checks one inline artifact captured inside a result
// payload before it is accepted.
func (policy Policy) ValidateInlineArtifact(field string, sizeBytes int64) []Violation {
	policy = policy.normalized()
	if sizeBytes <= policy.InlineArtifactMaxBytes {
		return nil
	}
	return []Violation{{
		Field:   field,
		Code:    "ARTIFACT_INLINE_SIZE_LIMIT_EXCEEDED",
		Message: "The inline artifact exceeds the configured size limit.",
		Limit:   policy.InlineArtifactMaxBytes,
		Actual:  sizeBytes,
	}}
}

// ValidateCollection checks the artifact count captured inside one result
// payload.
func (policy Policy) ValidateCollection(field string, count int) []Violation {
	policy = policy.normalized()
	if count <= policy.CollectionMaxItems {
		return nil
	}
	return []Violation{{
		Field:   field,
		Code:    "ARTIFACT_COLLECTION_LIMIT_EXCEEDED",
		Message: "The result payload contains too many artifacts.",
		Limit:   int64(policy.CollectionMaxItems),
		Actual:  int64(count),
	}}
}

// RetentionDuration returns the default artifact retention window.
func (policy Policy) RetentionDuration() time.Duration {
	policy = policy.normalized()
	return time.Duration(policy.DefaultRetentionDays) * 24 * time.Hour
}

func (policy Policy) normalized() Policy {
	defaults := DefaultPolicy()
	if policy.MaxArtifactBytes <= 0 {
		policy.MaxArtifactBytes = defaults.MaxArtifactBytes
	}
	if policy.InlineArtifactMaxBytes <= 0 {
		policy.InlineArtifactMaxBytes = defaults.InlineArtifactMaxBytes
	}
	if policy.CollectionMaxItems <= 0 {
		policy.CollectionMaxItems = defaults.CollectionMaxItems
	}
	if policy.DefaultRetentionDays <= 0 {
		policy.DefaultRetentionDays = defaults.DefaultRetentionDays
	}
	if len(policy.AllowedContentTypes) == 0 {
		policy.AllowedContentTypes = defaults.AllowedContentTypes
	}
	return policy
}

func required(field string) Violation {
	return Violation{
		Field:   field,
		Code:    "ARTIFACT_FIELD_REQUIRED",
		Message: "The field is required.",
	}
}
