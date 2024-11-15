package repository

import (
	"log"

	"github.com/doug-martin/goqu/v9"
)

func (r *Repository) MoveSerializedItems(items []int, locationID int, status string) error {
	qb := goqu.Dialect("postgres")

	locationCase := goqu.Case()
	statusCase := goqu.Case()

	for _, item := range items {
		locationCase = locationCase.When(goqu.Ex{"id": item}, locationID)
		statusCase = statusCase.When(goqu.Ex{"id": item}, status)
	}

	// Build the update query
	query := qb.From("items").Update().
		Set(goqu.Record{
			"location_id": locationCase,
			"status":      statusCase,
		}).
		Where(goqu.C("id").In(items))

	sql, args, err := query.ToSQL()
	if err != nil {
		log.Fatalf("Failed to build query: %v", err)
	}

	// Execute the query
	_, execErr := r.DB.Exec(sql, args...)
	if execErr != nil {
		log.Fatalf("Failed to execute query: %v", execErr)
	}

	return err
}
