package origin

import "strings"

type Origin string

const (
	OriginDrugaEra Origin = "druga-era"
	OriginProbis   Origin = "probis"
	OriginPersonal Origin = "personal"
)

func (o Origin) IsValid() bool {
	switch o {
	case OriginDrugaEra, OriginProbis, OriginPersonal:
		return true
	default:
		return false
	}
}

func NewOrigin(value string) Origin {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return Origin(normalized)
}

func (o Origin) ContainsKeyword(keyword string) bool {
	return strings.Contains(string(o), keyword)
}

func (o Origin) isPredefined() bool {
	var predefinedOrigins = map[Origin]bool{
		OriginDrugaEra: true,
		OriginProbis:   true,
		OriginPersonal: true,
	}

	return predefinedOrigins[o]
}
