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
	str := c.Name
	if len(str) < 3 {
		str = str + strings.Repeat("x", 3-len(str))
	}
	str = str[:3]
	c.PyrID = strings.ToUpper(str) + "1"
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
