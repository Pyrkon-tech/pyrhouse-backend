package models

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

type ItemCategory struct {
	ID    int    `json:"id,omitempty" db:"category_id"`
	Name  string `json:"name,omitempty" db:"type"`
	Label string `json:"label,omitempty" binding:"required" db:"label"`
	PyrID string `json:"pyr_id" binding:"omitempty,alphanum,min=1,max=4" db:"pyr_id"`
	Type  string `json:"type" binding:"alphanum,min=1,max=24" db:"category_type"`
}

func (c *ItemCategory) GenerateNameFromLabel() {
	if c.Name == "" && c.Label != "" {
		c.Name = strings.ToLower(removeDiacritics(c.Label))
	}
}

func (c *ItemCategory) GeneratePyrID() {
	if c.PyrID != "" {
		return
	}

	// Sprawdzamy czy w label występują znane skróty (3-4 litery wielkie)
	words := strings.Fields(c.Label)
	for _, word := range words {
		if len(word) >= 3 && len(word) <= 4 && strings.ToUpper(word) == word {
			c.PyrID = word
			return
		}
	}

	// Jeśli nie znaleziono skrótu, generujemy standardowo
	str := c.Name
	words = strings.Fields(str)

	if len(words) >= 3 {
		// Jeśli mamy 3 lub więcej słów, bierzemy pierwszą literę z każdego słowa
		var builder strings.Builder
		for i := 0; i < 3; i++ {
			if i < len(words) {
				builder.WriteByte(words[i][0])
			}
		}
		c.PyrID = strings.ToUpper(builder.String())
	} else {
		// Standardowa logika dla mniej niż 3 słów
		if len(str) < 3 {
			str = str + strings.Repeat("x", 3-len(str))
		}
		str = str[:3]
		c.PyrID = strings.ToUpper(str)
	}
}

func removeDiacritics(input string) string {
	t := norm.NFD.String(input)

	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, t)
}

func (i *ItemCategory) CreateLogView() AuditLog {
	return AuditLog{
		ResourceID:   i.ID,
		ResourceType: "category",
	}
}
