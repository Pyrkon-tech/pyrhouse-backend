package postgres_repository

import {
	"database/sql",
	"warehouse/pkg/models"
}

type Reposistory struct {
	DB *sql.DB
}

func (r *Repository) GetLocationItems() ([]models.Item){
	rows, err := h.DB.Query("SELECT id, item_type, item_serial FROM items WHERE location_id = $1", c.Param("id"))
	if err != nil {
		log.Fatal("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not insert location"})
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var item models.Item
		if err := rows.Scan(&item.ID, &item.Type, &item.Serial); err != nil {
			log.Fatal("Error executing SQL statement: ", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not insert location"})
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		log.Fatal("Error executing SQL statement: ", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not insert location"})
	}

	return items
}