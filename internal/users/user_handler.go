package users

import (
	"log"
	"net/http"
	"strconv"
	"warehouse/pkg/models"
	"warehouse/pkg/security"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UsersHandler struct {
	Repository UserRepository
}

func NewHandler(r UserRepository) *UsersHandler {
	return &UsersHandler{
		Repository: r,
	}
}

func (h *UsersHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/users", security.Authorize("admin"), h.RegisterUser)
	router.PATCH("/users/:id", security.Authorize("admin"), h.UpdateUser)
	router.GET("/users/:id", security.Authorize("user"), h.GetUser)
	router.GET("/users", security.Authorize("moderator"), h.GetUserList)
	router.POST("/users/:id/points", security.Authorize("admin"), h.AddUserPoints)
}

func (h *UsersHandler) RegisterUser(c *gin.Context) {
	var req models.CreateUserRequest
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

func (h *UsersHandler) UpdateUser(c *gin.Context) {
	var req models.UpdateUserRequest
	var err error

	if err = c.BindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		log.Fatal(err)
		return
	}

	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID", "details": err.Error()})
		return
	}

	if !h.isAllowed(c, userID, "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden", "details": "You are not allowed to access this resource"})
		return
	}

	user, err := h.Repository.GetUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Unable to find user", "details": err.Error(), "code": "USER_NOT_FOUND"})
		return
	}

	changes := &models.UserChanges{}

	if req.Password != nil && *req.Password != "" {
		if len(*req.Password) > 5 {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
				return
			}
			passwordHash := string(hashedPassword)
			changes.PasswordHash = &passwordHash
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 6 characters long"})
			return
		}
	}

	if req.Role != nil && *req.Role != user.Role {
		role := string(*req.Role)
		changes.Role = &role
	}

	if req.Points != nil {
		points := *req.Points
		changes.Points = &points
	}

	// Jeśli nie ma żadnych zmian, zwracamy sukces
	if !changes.HasChanges() {
		c.JSON(http.StatusOK, user)
		return
	}

	// Aktualizujemy użytkownika tylko jeśli są zmiany
	if err := h.Repository.UpdateUser(userID, changes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user", "details": err.Error()})
		return
	}

	// Pobieramy zaktualizowanego użytkownika
	updatedUser, err := h.Repository.GetUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get updated user", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}

func (h *UsersHandler) AddUserPoints(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID", "details": err.Error()})
		return
	}

	var req struct {
		Points int `json:"points" binding:"required"`
	}

	if err = c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	err = h.Repository.AddUserPoints(userID, req.Points)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add user points",
			"details": err.Error(),
		})
		return
	}

	user, err := h.Repository.GetUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get updated user",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User points updated successfully",
		"points":  user.Points,
	})
}

func (h *UsersHandler) GetUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID", "details": err.Error()})
		return
	}

	if !h.isAllowed(c, userID, "moderator") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden", "details": "You are not allowed to access this resource"})
		return
	}

	user, err := h.Repository.GetUser(userID)
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Unable to find user", "details": err.Error(), "code": "USER_NOT_FOUND"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user", "details": err.Error()})
	}

	c.JSON(http.StatusOK, user)
}

func (h *UsersHandler) GetUserList(c *gin.Context) {
	users, err := h.Repository.GetUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not obtain list of users", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

func (h *UsersHandler) isAllowed(c *gin.Context, userID int, userRole string) bool {
	authID, ok := c.Get("userID")
	authID, err := strconv.Atoi(authID.(string))
	if err != nil || authID == 0 || !ok {
		return false
	}

	if authID != userID && !security.IsAllowed(c, userRole) {
		return false
	}

	return true
}
