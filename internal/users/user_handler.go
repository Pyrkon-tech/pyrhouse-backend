package users

import (
	"database/sql"
	"log"
	"net/http"
	"warehouse/internal/repository/user"
	"warehouse/pkg/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UsersHandler struct {
	DB         *sql.DB
	Repository *user.UserRepository
}

func RegisterRoutes(router *gin.Engine, db *sql.DB, r *user.UserRepository) {
	handler := UsersHandler{DB: db, Repository: r}

	router.POST("/users", handler.RegisterUser)
	router.GET("/users", handler.GetUserList)
}

func (h *UsersHandler) RegisterUser(c *gin.Context) {
	var req models.UserRequest
	if err := c.BindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		log.Fatal(err)
		return
	}
	log.Println(req.Username)

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	err = h.Repository.PersistUser(req, hashedPassword)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create user",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}

func (h *UsersHandler) GetUser(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, gin.H{"message": "Not implemented"})
}

func (h *UsersHandler) GetUserList(c *gin.Context) {
	users, err := h.Repository.GetUsers()

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not obtain list of users", "details": err.Error()})
	}

	c.JSON(http.StatusOK, users)
}
