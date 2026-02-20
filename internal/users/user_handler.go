package users

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"warehouse/internal/models"
	"warehouse/internal/roles"
	"warehouse/internal/security"

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
	router.PATCH("/users/:id", security.Authorize("user"), h.UpdateUser)
	router.GET("/users/:id", security.Authorize("user"), h.GetUser)
	router.GET("/users", security.Authorize("moderator"), h.GetUserList)
	router.POST("/users/:id/points", security.Authorize("admin"), h.AddUserPoints)
	router.DELETE("/users/:id", security.Authorize("admin"), h.DeleteUser)
}

func (h *UsersHandler) RegisterPublicRoutes(router *gin.Engine) {
	router.POST("/users/register", h.RegisterPublicUser)
}

func (h *UsersHandler) RegisterPublicUser(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data", "details": err.Error()})
		return
	}

	req.Active = false
	userRole := roles.Role("user")
	req.Role = &userRole
	err := h.createUser(req)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}

func (h *UsersHandler) RegisterUser(c *gin.Context) {

	var req models.CreateUserRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input data", "details": err.Error()})
		return
	}

	req.Active = true

	err := h.createUser(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}

func (h *UsersHandler) createUser(req models.CreateUserRequest) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if req.Role == nil {
		defaultRole := roles.Role("user")
		req.Role = &defaultRole
	}

	err = h.Repository.PersistUser(req, hashedPassword)
	if err != nil {
		return err
	}

	return nil
}

type UpdateUserContext struct {
	c           *gin.Context
	userID      int
	req         *models.UpdateUserRequest
	user        *models.User
	changes     *models.UserChanges
	isOwner     bool
	isAdmin     bool
	isModerator bool
}

func (h *UsersHandler) UpdateUser(c *gin.Context) {
	ctx, err := h.prepareUpdateContext(c)
	if err != nil {
		return
	}

	if err := h.validateAndApplyChanges(ctx); err != nil {
		return
	}

	if !ctx.changes.HasChanges() {
		c.JSON(http.StatusOK, ctx.user)
		return
	}

	if err := h.Repository.UpdateUser(ctx.userID, ctx.changes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating user", "details": err.Error()})
		return
	}

	updatedUser, err := h.Repository.GetUser(ctx.userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching updated user", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}

func (h *UsersHandler) prepareUpdateContext(c *gin.Context) (*UpdateUserContext, error) {
	var req models.UpdateUserRequest
	if err := c.BindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid input data", "details": err.Error()})
		return nil, err
	}

	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID", "details": err.Error()})
		return nil, err
	}

	user, err := h.Repository.GetUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found", "details": err.Error(), "code": "USER_NOT_FOUND"})
		return nil, err
	}

	authID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return nil, fmt.Errorf("userID not found in context")
	}
	authIDStr, ok := authID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
		return nil, fmt.Errorf("userID is not a string")
	}
	authIDInt, err := strconv.Atoi(authIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
		return nil, fmt.Errorf("invalid userID format: %w", err)
	}

	return &UpdateUserContext{
		c:           c,
		userID:      userID,
		req:         &req,
		user:        user,
		changes:     &models.UserChanges{},
		isOwner:     authIDInt == userID,
		isAdmin:     security.IsAllowed(c, "admin"),
		isModerator: security.IsAllowed(c, "moderator"),
	}, nil
}

func (h *UsersHandler) validateAndApplyChanges(ctx *UpdateUserContext) error {
	if err := h.validatePasswordChange(ctx); err != nil {
		return err
	}

	if err := h.validateRoleChange(ctx); err != nil {
		return err
	}

	if err := h.validateFullnameChange(ctx); err != nil {
		return err
	}

	if err := h.validatePointsChange(ctx); err != nil {
		return err
	}

	if err := h.validateUsernameChange(ctx); err != nil {
		return err
	}

	if err := h.validateActiveChange(ctx); err != nil {
		return err
	}

	return nil
}

func (h *UsersHandler) validatePasswordChange(ctx *UpdateUserContext) error {
	if ctx.req.Password == nil || *ctx.req.Password == "" {
		return nil
	}

	if !ctx.isOwner && !ctx.isAdmin {
		ctx.c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden", "details": "Only account owner or admin can change password"})
		return fmt.Errorf("unauthorized password change")
	}

	if len(*ctx.req.Password) < 6 {
		ctx.c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 6 characters"})
		return fmt.Errorf("password too short")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*ctx.req.Password), bcrypt.DefaultCost)
	if err != nil {
		ctx.c.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
		return err
	}

	passwordHash := string(hashedPassword)
	ctx.changes.PasswordHash = &passwordHash
	return nil
}

func (h *UsersHandler) validateRoleChange(ctx *UpdateUserContext) error {
	if ctx.req.Role == nil || *ctx.req.Role == ctx.user.Role {
		return nil
	}

	if !ctx.isAdmin {
		ctx.c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden", "details": "Only admin can change user role"})
		return fmt.Errorf("unauthorized role change")
	}

	role := string(*ctx.req.Role)
	ctx.changes.Role = &role
	return nil
}

func (h *UsersHandler) validateFullnameChange(ctx *UpdateUserContext) error {
	if ctx.req.Fullname == nil {
		return nil
	}

	if !ctx.isOwner && !ctx.isModerator {
		ctx.c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden", "details": "Only account owner or moderator can change fullname"})
		return fmt.Errorf("unauthorized fullname change")
	}

	if *ctx.req.Fullname == "" {
		ctx.c.JSON(http.StatusBadRequest, gin.H{"error": "Fullname cannot be empty"})
		return fmt.Errorf("empty fullname")
	}

	if ctx.user.Fullname == nil || *ctx.req.Fullname != *ctx.user.Fullname {
		ctx.changes.Fullname = ctx.req.Fullname
	}
	return nil
}

func (h *UsersHandler) validatePointsChange(ctx *UpdateUserContext) error {
	if ctx.req.Points == nil {
		return nil
	}

	if !ctx.isModerator {
		ctx.c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden", "details": "Only moderator can change user points"})
		return fmt.Errorf("unauthorized points change")
	}

	points := *ctx.req.Points
	ctx.changes.Points = &points
	return nil
}

func (h *UsersHandler) validateUsernameChange(ctx *UpdateUserContext) error {
	if ctx.req.Username == nil || *ctx.req.Username == "" {
		return nil
	}

	if !ctx.isOwner && !ctx.isAdmin {
		ctx.c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden", "details": "Only account owner or admin can change username"})
		return fmt.Errorf("unauthorized username change")
	}

	if *ctx.req.Username == "" {
		ctx.c.JSON(http.StatusBadRequest, gin.H{"error": "Username cannot be empty"})
		return fmt.Errorf("empty username")
	}

	isUnique, err := h.Repository.IsUsernameUnique(*ctx.req.Username)
	if err != nil {
		ctx.c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking username uniqueness", "details": err.Error()})
	}
	if !isUnique {
		ctx.c.JSON(http.StatusConflict, gin.H{"error": "Username already taken", "details": "Username already taken"})
		return fmt.Errorf("username already exists")
	}

	ctx.changes.Username = ctx.req.Username
	return nil
}

func (h *UsersHandler) validateActiveChange(ctx *UpdateUserContext) error {
	if ctx.req.Active == nil {
		return nil
	}

	switch true {
	case ctx.isAdmin:
		active := *ctx.req.Active
		ctx.changes.Active = &active
		return nil
	case ctx.isModerator && ctx.user.Role != "user":
		ctx.c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden", "details": "Cannot change active status of user with role other than user"})
		return fmt.Errorf("unauthorized active change")
	case !ctx.isAdmin && !ctx.isModerator:
		ctx.c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden", "details": "Only admin or moderator can change user active status"})
		return fmt.Errorf("unauthorized active change")
	}

	active := *ctx.req.Active
	ctx.changes.Active = &active
	return nil
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

	if !security.IsOwnerOrAllowed(c, userID, "moderator") {
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

func (h *UsersHandler) DeleteUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID", "details": err.Error()})
		return
	}

	if !security.IsOwnerOrAllowed(c, userID, "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden", "details": "You don't have permission to perform this operation"})
		return
	}

	err = h.Repository.DeleteUser(userID)
	if err != nil {
		if strings.Contains(err.Error(), "cannot delete user, has assigned transfers") {
			c.JSON(http.StatusConflict, gin.H{"error": "Cannot delete user", "details": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot delete user", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
