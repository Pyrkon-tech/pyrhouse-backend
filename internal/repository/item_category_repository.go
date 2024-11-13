package repository

import (
	"log"
	"warehouse/pkg/models"
)

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
		log.Fatal("failed to delete item category: ", err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal("could not retrieve rows affected: ", err)
		return err
	}

	if rowsAffected == 0 {
		log.Fatal("no item category found with id: ", CategoryID)
		return err
	}

	return nil
}
