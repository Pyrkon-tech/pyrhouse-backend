package repository

import (
	"log"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
)

func (r *Repository) GetCategories() (*[]models.ItemCategory, error) {
	qb := goqu.Dialect("postgres")
	query := qb.Select("id", "item_category", "label").From("item_category")

	// rows, err := r.Query(query)
	sql, args, err := query.ToSQL()

	if err != nil {
		log.Fatalf("Failed to build query: %v", err)
	}
	rows, err := r.DB.Query(sql, args...)
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}
	defer rows.Close()

	var itemCategories []models.ItemCategory
	for rows.Next() {
		var itemCategory models.ItemCategory
		if err := rows.Scan(&itemCategory.ID, &itemCategory.Type, &itemCategory.Label); err != nil {
			return nil, err
		}
		itemCategories = append(itemCategories, itemCategory)
	}
	return &itemCategories, err
}

func (r *Repository) PersistItemCategory(itemCategory models.ItemCategory) (*models.ItemCategory, error) {
	stmtString := "INSERT INTO item_category (item_category, label) VALUES ($1, $2)"
	stmt, err := r.DB.Prepare(stmtString)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	err = r.DB.QueryRow(
		stmtString+" RETURNING id",
		itemCategory.Type,
		itemCategory.Label,
	).Scan(&itemCategory.ID)

	return &itemCategory, err
}

func (r *Repository) DeleteItemCategoryByID(CategoryID string) error {
	query := `DELETE FROM item_category WHERE id = $1`
	result, err := r.DB.Exec(query, CategoryID)
	if err != nil {
		log.Fatal("failed to delete asset category: ", err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal("could not retrieve rows affected: ", err)
		return err
	}

	if rowsAffected == 0 {
		log.Fatal("no asset category found with id: ", CategoryID)
		return err
	}

	return nil
}
