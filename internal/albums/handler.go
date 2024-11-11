package albums

import (
	"database/sql"
	"net/http"

	"warehouse/pkg/models"

	"github.com/gin-gonic/gin"
)

type AlbumHandler struct {
	DB *sql.DB
}

func RegisterRoutes(router *gin.Engine, db *sql.DB) {
	handler := AlbumHandler{DB: db}
	router.GET("/albums", handler.GetAlbums)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	// Add more routes here (e.g., POST /albums)
}

func (h *AlbumHandler) GetAlbums(c *gin.Context) {
	var albums []models.Album

	rows, err := h.DB.Query("SELECT id, title, artist, price FROM albums")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var album models.Album
		if err := rows.Scan(&album.ID, &album.Title, &album.Artist, &album.Price); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning database"})
			return
		}
		albums = append(albums, album)
	}

	c.JSON(http.StatusOK, albums)
}
