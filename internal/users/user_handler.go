package users

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"warehouse/pkg/models"
	"warehouse/pkg/roles"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowe dane wejściowe", "details": err.Error()})
		return
	}

	req.Active = false
	userRole := roles.Role("user")
	req.Role = &userRole
	err := h.createUser(req)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nie udało się utworzyć użytkownika", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Użytkownik zarejestrowany pomyślnie"})
}

func (h *UsersHandler) RegisterUser(c *gin.Context) {

	var req models.CreateUserRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowe dane wejściowe", "details": err.Error()})
		return
	}

	req.Active = true

	err := h.createUser(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nie udało się utworzyć użytkownika", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Użytkownik zarejestrowany pomyślnie"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Błąd podczas aktualizacji użytkownika", "details": err.Error()})
		return
	}

	updatedUser, err := h.Repository.GetUser(ctx.userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Błąd podczas pobierania zaktualizowanego użytkownika", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}

func (h *UsersHandler) prepareUpdateContext(c *gin.Context) (*UpdateUserContext, error) {
	var req models.UpdateUserRequest
	if err := c.BindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowe dane wejściowe", "details": err.Error()})
		return nil, err
	}

	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowe ID użytkownika", "details": err.Error()})
		return nil, err
	}

	user, err := h.Repository.GetUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Nie znaleziono użytkownika", "details": err.Error(), "code": "USER_NOT_FOUND"})
		return nil, err
	}

	authID, _ := c.Get("userID")
	authIDInt, _ := strconv.Atoi(authID.(string))

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
		ctx.c.JSON(http.StatusForbidden, gin.H{"error": "Brak dostępu", "details": "Tylko właściciel konta lub administrator może zmienić hasło"})
		return fmt.Errorf("unauthorized password change")
	}

	if len(*ctx.req.Password) < 6 {
		ctx.c.JSON(http.StatusBadRequest, gin.H{"error": "Hasło musi mieć co najmniej 6 znaków"})
		return fmt.Errorf("password too short")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*ctx.req.Password), bcrypt.DefaultCost)
	if err != nil {
		ctx.c.JSON(http.StatusInternalServerError, gin.H{"error": "Błąd podczas hashowania hasła"})
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
		ctx.c.JSON(http.StatusForbidden, gin.H{"error": "Brak dostępu", "details": "Tylko administrator może zmienić rolę użytkownika"})
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
		ctx.c.JSON(http.StatusForbidden, gin.H{"error": "Brak dostępu", "details": "Tylko właściciel konta lub moderator może zmienić imię i nazwisko"})
		return fmt.Errorf("unauthorized fullname change")
	}

	if *ctx.req.Fullname == "" {
		ctx.c.JSON(http.StatusBadRequest, gin.H{"error": "Imię i nazwisko nie może być puste"})
		return fmt.Errorf("empty fullname")
	}

	if *ctx.req.Fullname != ctx.user.Fullname {
		ctx.changes.Fullname = ctx.req.Fullname
	}
	return nil
}

func (h *UsersHandler) validatePointsChange(ctx *UpdateUserContext) error {
	if ctx.req.Points == nil {
		return nil
	}

	if !ctx.isModerator {
		ctx.c.JSON(http.StatusForbidden, gin.H{"error": "Brak dostępu", "details": "Tylko moderator może zmienić punkty użytkownika"})
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
		ctx.c.JSON(http.StatusForbidden, gin.H{"error": "Brak dostępu", "details": "Tylko właściciel konta lub administrator może zmienić nazwę użytkownika"})
		return fmt.Errorf("unauthorized username change")
	}

	if *ctx.req.Username == "" {
		ctx.c.JSON(http.StatusBadRequest, gin.H{"error": "Nazwa użytkownika nie może być pusta"})
		return fmt.Errorf("empty username")
	}

	isUnique, err := h.Repository.IsUsernameUnique(*ctx.req.Username)
	if err != nil {
		ctx.c.JSON(http.StatusInternalServerError, gin.H{"error": "Błąd podczas sprawdzania unikalności nazwy użytkownika", "details": err.Error()})
	}
	if !isUnique {
		ctx.c.JSON(http.StatusConflict, gin.H{"error": "Nazwa użytkownika jest już zajęta", "details": "Nazwa użytkownika jest już zajęta"})
		return fmt.Errorf("username already exists")
	}

	ctx.changes.Username = ctx.req.Username
	return nil
}

func (h *UsersHandler) validateActiveChange(ctx *UpdateUserContext) error {
	if ctx.req.Active == nil {
		return nil
	}

	if !ctx.isAdmin {
		ctx.c.JSON(http.StatusForbidden, gin.H{"error": "Brak dostępu", "details": "Tylko administrator może zmienić aktywność użytkownika"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowe ID użytkownika", "details": err.Error()})
		return
	}

	if !security.IsOwnerOrAllowed(c, userID, "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Brak dostępu", "details": "Nie masz uprawnień do wykonania tej operacji"})
		return
	}

	err = h.Repository.DeleteUser(userID)
	if err != nil {
		if strings.Contains(err.Error(), "nie można usunąć użytkownika, ponieważ ma przypisane transfery") {
			c.JSON(http.StatusConflict, gin.H{"error": "Nie można usunąć użytkownika", "details": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nie można usunąć użytkownika", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Użytkownik został usunięty"})
}
