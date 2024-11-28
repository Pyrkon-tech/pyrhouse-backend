package users

import (
	"database/sql"
	"log"
	"net/http"
	"warehouse/internal/repository/user"
	"warehouse/pkg/models"
	"warehouse/pkg/security"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UsersHandler struct {
	Repository *user.UserRepository
}

func RegisterRoutes(router *gin.RouterGroup, db *sql.DB, r *user.UserRepository) {
	handler := UsersHandler{Repository: r}

	router.POST("/users", security.Authorize("admin"), handler.RegisterUser)
	router.GET("/users", security.Authorize("moderator"), handler.GetUserList)
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
	// TODO ?
	c.JSON(http.StatusInternalServerError, gin.H{"message": "Not implemented"})
}

func (h *UsersHandler) GetUserList(c *gin.Context) {
	users, err := h.Repository.GetUsers()

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not obtain list of users", "details": err.Error()})
	}

	c.JSON(http.StatusOK, users)
}
