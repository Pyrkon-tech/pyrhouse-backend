package origins

import (
	"context"
	"fmt"
	"strings"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// OriginResolution is the result of validating an origin string, ready for DB storage.
type OriginResolution struct {
	OriginID     int
	OriginSuffix *string // nil for plain origins, "jan" for "personal-jan"
	DisplayValue string  // "druga-era" or "personal-jan" — for API responses
}

// ResolveOrigin validates and parses an origin string into origin_id + suffix.
// Replaces metadata.NewOrigin().
func (s *Service) ResolveOrigin(ctx context.Context, value string) (*OriginResolution, error) {
	normalized := normalizeOrigin(value)

	// 1. Exact match — slug exists and is active
	if origin, _ := s.repo.GetBySlug(ctx, normalized); origin != nil && origin.Active {
		return &OriginResolution{OriginID: origin.ID, DisplayValue: normalized}, nil
	}

	// 2. Suffix match — "personal-jan" → look for slug "personal" with allow_suffix=true
	if idx := strings.Index(normalized, "-"); idx > 0 {
		base := normalized[:idx]
		suffix := normalized[idx+1:]
		if origin, _ := s.repo.GetBySlug(ctx, base); origin != nil && origin.AllowSuffix && origin.Active {
			return &OriginResolution{
				OriginID:     origin.ID,
				OriginSuffix: &suffix,
				DisplayValue: normalized,
			}, nil
		}
	}

	return nil, fmt.Errorf("invalid origin: %s", value)
}

// FormatOriginDisplay reconstructs the display string from origin slug + suffix.
func FormatOriginDisplay(slug string, suffix *string) string {
	if suffix != nil && *suffix != "" {
		return slug + "-" + *suffix
	}
	return slug
}

func normalizeOrigin(value string) string {
	return strings.Replace(strings.ToLower(strings.TrimSpace(value)), " ", "-", -1)
}
