package users

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UsersHandler struct {
	DB *sql.DB
}

func RegisterRoutes(router *gin.Engine, db *sql.DB) {
	handler := UsersHandler{DB: db}

	router.POST("/users", handler.RegisterUser)
	// router.GET("/locations", handler.GetLocations)
}

func (h *UsersHandler) RegisterUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Fullname string `json:"fullname"`
		Role     string `json:"role" binding:"required"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		log.Fatal(err)
		return
	}
	log.Println(req.Username)

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Insert user into the database
	// Add handling for optional paramenter
	_, err = h.DB.Exec("INSERT INTO users (username, password_hash, fullname, role) VALUES ($1, $2, $3, $4)",
		req.Username, string(hashedPassword), req.Fullname, req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}

func (h *UsersHandler) GetUser(c *gin.Context) {

}
