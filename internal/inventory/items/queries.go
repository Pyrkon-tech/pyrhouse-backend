package items

import (
	"github.com/doug-martin/goqu/v9"
)

// RetrieveItemQuery reprezentuje zapytanie o pojedynczy element
type RetrieveItemQuery struct {
	ID           *int   `uri:"id" binding:"required,number"`
	CategoryType string `uri:"category" binding:"required"`
}

// RetrieveItemListQuery reprezentuje zapytanie o listę elementów
type RetrieveItemListQuery struct {
	LocationIDs   []int  `form:"location_ids" binding:"omitempty"`
	CategoryID    *int   `form:"category_id" binding:"omitempty,number"`
	CategoryType  string `form:"category_type"`
	CategoryLabel string `form:"category_label"`
}

// AddCondition dodaje warunek do zapytania
func (q *RetrieveItemListQuery) AddCondition(key string, value interface{}) {
	switch key {
	case "location_ids":
		if ids, ok := value.([]int); ok {
			q.LocationIDs = ids
		}
	case "category_id":
		if id, ok := value.(int); ok {
			q.CategoryID = &id
		}
	case "category_label":
		if label, ok := value.(string); ok {
			q.CategoryLabel = label
		}
	}
}

// BuildConditions buduje warunki zapytania
func (q *RetrieveItemListQuery) BuildConditions(aliases map[string]string) goqu.Ex {
	conditions := goqu.Ex{}

	if q.LocationIDs != nil {
		conditions[aliases["location_ids"]] = q.LocationIDs
	}
	if q.CategoryID != nil {
		conditions[aliases["category_id"]] = *q.CategoryID
	}
	if q.CategoryLabel != "" {
		conditions[aliases["category_label"]] = q.CategoryLabel
	}

	return conditions
}

// HasConditions sprawdza, czy zapytanie ma warunki
func (q *RetrieveItemListQuery) HasConditions() bool {
	return len(q.LocationIDs) > 0 || q.CategoryID != nil || q.CategoryLabel != ""
}
