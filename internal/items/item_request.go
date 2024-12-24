package items

import (
	"github.com/doug-martin/goqu/v9"
)

type fetchItemsQuery struct {
	LocationIDs   []int  `form:"location_ids" binding:"omitempty"`
	CategoryID    *int   `form:"category_id" binding:"omitempty,number"`
	CategoryType  string `form:"category_type"`
	CategoryLabel string `form:"category_label"`
}

func (q *fetchItemsQuery) BuildConditions(aliases map[string]string) goqu.Ex {
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
