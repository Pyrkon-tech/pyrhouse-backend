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

	// Check if the label contains known abbreviations (3-4 uppercase letters)
	words := strings.Fields(c.Label)
	for _, word := range words {
		if len(word) >= 3 && len(word) <= 4 && strings.ToUpper(word) == word {
			c.PyrID = word
			return
		}
	}

	// If no abbreviation was found, generate one using the standard method
	str := c.Name
	words = strings.Fields(str)

	if len(words) >= 3 {
		// If we have 3 or more words, take the first letter from each word
		var builder strings.Builder
		for i := 0; i < 3; i++ {
			if i < len(words) {
				builder.WriteByte(words[i][0])
			}
		}
		c.PyrID = strings.ToUpper(builder.String())
	} else {
		// Standard logic for fewer than 3 words
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
