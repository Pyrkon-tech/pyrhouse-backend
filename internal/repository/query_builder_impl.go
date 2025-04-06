package repository

import (
	"github.com/doug-martin/goqu/v9"
)

type queryBuilderImpl struct {
	conditions map[string]interface{}
}

func NewQueryBuilder() QueryBuilder {
	return &queryBuilderImpl{
		conditions: make(map[string]interface{}),
	}
}

func (q *queryBuilderImpl) AddCondition(key string, value interface{}) {
	q.conditions[key] = value
}

func (q *queryBuilderImpl) BuildConditions(aliases map[string]string) goqu.Ex {
	conditions := goqu.Ex{}
	for key, value := range q.conditions {
		if alias, ok := aliases[key]; ok {
			conditions[alias] = value
		} else {
			conditions[key] = value
		}
	}
	return conditions
}
