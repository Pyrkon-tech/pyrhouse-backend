package repository

import (
	"context"
	"fmt"

	"github.com/doug-martin/goqu/v9"
)

// CountByStatus counts the rows of table per value of its status column in one grouped query.
// Every status in statuses is present in the result, zero when no row has it.
func CountByStatus(ctx context.Context, r *Repository, table string, statuses []string) (map[string]int, error) {
	counts := make(map[string]int, len(statuses))
	for _, s := range statuses {
		counts[s] = 0
	}

	var rows []struct {
		Status string `db:"status"`
		Count  int    `db:"count"`
	}
	err := r.GoquDBWrapper.
		From(table).
		Select(goqu.C("status"), goqu.COUNT("*").As("count")).
		GroupBy(goqu.C("status")).
		Executor().ScanStructsContext(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("count %s by status: %w", table, err)
	}
	for _, row := range rows {
		counts[row.Status] = row.Count
	}
	return counts, nil
}
