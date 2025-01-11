package metadata

import (
	"fmt"
	"strings"
)

type Origin string

const (
	OriginDrugaEra Origin = "druga-era"
	OriginProbis   Origin = "probis"
	OriginTargi    Origin = "targowe"
	OriginPersonal Origin = "personal"
	OriginOther    Origin = "other"
)

func (o Origin) IsValid() bool {
	switch o {
	case OriginDrugaEra, OriginProbis, OriginTargi:
		return true
	default:
		return false
	}
}

func (o Origin) isPredefined() bool {
	if o.ContainsKeyword(string(OriginPersonal)) {
		return true
	} else if o.ContainsKeyword(string(OriginOther)) {
		return true
	}
	return false
}

func NewOrigin(value string) (Origin, error) {
	normalized := strings.Replace(strings.ToLower(strings.TrimSpace(value)), " ", "-", -1)
	origin := Origin(normalized)
	if !origin.IsValid() && !origin.isPredefined() {
		return origin, fmt.Errorf(
			"value not valid, only valid values are: %s, %s, %s, %s, %s",
			OriginDrugaEra, OriginProbis, OriginTargi, OriginPersonal, OriginOther,
		)
	}

	return origin, nil
}

func (o Origin) String() string {
	return string(o)
}

func (o Origin) ContainsKeyword(keyword string) bool {
	return strings.Contains(string(o), keyword)
}
