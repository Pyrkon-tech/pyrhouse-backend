package items

import "github.com/doug-martin/goqu/v9"

type QueryBuilder interface {
	BuildConditions(aliases map[string]string) goqu.Ex
}
