package items

import (
	"github.com/doug-martin/goqu/v9"
)

type retrieveItemQuery struct {
	ID           *int   `uri:"id" binding:"required,number"`
	CategoryType string `uri:"category" binding:"required"`
}

type retrieveItemListQuery struct {
	LocationIDs   []int  `form:"location_ids" binding:"omitempty"`
	CategoryID    *int   `form:"category_id" binding:"omitempty,number"`
	CategoryType  string `form:"category_type"`
	CategoryLabel string `form:"category_label"`
}

func (q *retrieveItemListQuery) AddCondition(key string, value interface{}) {
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

func (q *retrieveItemListQuery) BuildConditions(aliases map[string]string) goqu.Ex {
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

func (q *retrieveItemListQuery) HasConditions() bool {
	return len(q.LocationIDs) > 0 || q.CategoryID != nil || q.CategoryLabel != ""
}
