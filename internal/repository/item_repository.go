package repository

import (
	"log"
	"warehouse/pkg/models"
)

func (r *Repository) HasRelatedItems(categoryID string) bool {
	query := `SELECT COUNT(*) FROM items WHERE item_category_id = $1`
	var count int
	err := r.DB.QueryRow(query, categoryID).Scan(&count)
	if err != nil {
		log.Fatal("failed to check related items: ", err)

		return false
	}
	return count > 0
}

func (r *Repository) PersistItem(itemRequest models.ItemRequest) (*models.Item, error) {
	stmtString := "INSERT INTO items (item_serial, location_id, item_category_id) VALUES ($1, $2, $3)"
	stmt, err := r.DB.Prepare(stmtString)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	var item models.Item
	err = r.DB.QueryRow(
		stmtString+" RETURNING id, item_serial, location_id, item_category_id",
		itemRequest.Serial,
		itemRequest.LocationId,
		itemRequest.CategoryId,
	).Scan(&item.ID, &item.Serial, &item.Location.ID, &item.Category.ID)

	return &item, err
}
