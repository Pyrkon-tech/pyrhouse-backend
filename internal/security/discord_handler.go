package security

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"strconv"
	"warehouse/internal/models"
	"warehouse/internal/oauth"
	"warehouse/internal/roles"

	"github.com/gin-gonic/gin"
)

// DiscordUserRepository defines the repository methods required by DiscordHandler
type DiscordUserRepository interface {
	GetUser(id int) (*models.User, error)
	FindUserByDiscordID(discordID string) (*models.User, error)
	FindUserByUsername(username string) (*models.User, error)
	CreateDiscordUser(user *models.User) (*models.User, error)
	UpdateDiscordInfo(userID int, username string, avatarURL string) error
	LinkDiscord(userID int, discordID, discordUsername, avatarURL string) error
	MergeDiscordAccount(targetID, sourceID int) (sourceDeleted bool, err error)
}

type DiscordHandler struct {
	oauth    *oauth.DiscordOAuth
	userRepo DiscordUserRepository
}

func NewDiscordHandler(oauth *oauth.DiscordOAuth, userRepo DiscordUserRepository) *DiscordHandler {
	return &DiscordHandler{
		oauth:    oauth,
		userRepo: userRepo,
	}
}

func (h *DiscordHandler) DiscordLogin(c *gin.Context) {
	state := generateState()

	// Auto-detect HTTPS (works behind DigitalOcean/nginx reverse proxies)
	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		MaxAge:   600,
		Path:     "/",
		Secure:   isSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	c.Redirect(http.StatusTemporaryRedirect, h.oauth.GetAuthURL(state))
}

func (h *DiscordHandler) DiscordCallback(c *gin.Context) {
	frontendURL := h.oauth.GetFrontendURL()

	// Helper for redirecting with an error
	redirectWithError := func(errorMsg string) {
		if frontendURL != "" {
			redirectURL := frontendURL + "?error=" + url.QueryEscape(errorMsg)
			c.Redirect(http.StatusTemporaryRedirect, redirectURL)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": errorMsg})
		}
	}

	// 1. Verify state (CSRF protection)
	state := c.Query("state")
	savedState, _ := c.Cookie("oauth_state")
	if state != savedState {
		redirectWithError("Invalid state parameter")
		return
	}

	// 2. Get the code
	code := c.Query("code")
	if code == "" {
		redirectWithError("Missing authorization code")
		return
	}

	// 3. Exchange code for token
	token, err := h.oauth.ExchangeCode(code)
	if err != nil {
		redirectWithError("Failed to exchange code")
		return
	}

	// 4. Fetch user data from Discord
	discordUser, err := h.oauth.GetUser(token.AccessToken)
	if err != nil {
		redirectWithError("Failed to get Discord user")
		return
	}

	// 5. Find or create user
	user, err := h.findOrCreateUser(discordUser)
	if err != nil {
		redirectWithError("Failed to process user")
		return
	}

	// 6. Check if the account is active
	if !user.Active {
		redirectWithError("Account is inactive. Contact the admin and login again.")
		return
	}

	// 7. Generate JWT
	jwtToken, err := GenerateJWT(strconv.Itoa(user.ID), string(user.Role), user.Username)
	if err != nil {
		redirectWithError("Failed to generate token")
		return
	}

	// 8. Clear state cookie — no longer needed after successful auth
	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Secure:   isSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// 9. Redirect to frontend with token
	if frontendURL != "" {
		redirectURL := frontendURL + "?token=" + jwtToken
		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
	} else {
		// Fallback - return JSON if frontendURL is not set
		c.JSON(http.StatusOK, gin.H{"token": jwtToken})
	}
}

func (h *DiscordHandler) findOrCreateUser(discordUser *oauth.DiscordUser) (*models.User, error) {
	// Search for user by discord_id
	user, err := h.userRepo.FindUserByDiscordID(discordUser.ID)
	if err != nil {
		return nil, err
	}

	if user != nil {
		// User exists - update Discord data (username, avatar may have changed)
		avatarURL := h.oauth.GetAvatarURL(discordUser)
		if err := h.userRepo.UpdateDiscordInfo(user.ID, discordUser.Username, avatarURL); err != nil {
			// Log the error but continue
		}
		return user, nil
	}

	// Check if an account with the same username exists (without Discord link)
	existingUser, err := h.userRepo.FindUserByUsername(discordUser.Username)
	if err != nil {
		return nil, err
	}
	if existingUser != nil && existingUser.DiscordID == nil {
		// Account with this name exists - automatically link with Discord
		avatarURL := h.oauth.GetAvatarURL(discordUser)
		if err := h.userRepo.LinkDiscord(existingUser.ID, discordUser.ID, discordUser.Username, avatarURL); err != nil {
			return nil, err
		}
		// Get the updated user
		existingUser.DiscordID = &discordUser.ID
		existingUser.DiscordUsername = &discordUser.Username
		existingUser.AvatarURL = &avatarURL
		return existingUser, nil
	}

	// Create a new user
	avatarURL := h.oauth.GetAvatarURL(discordUser)
	newUser := &models.User{
		Username:        discordUser.Username,
		DiscordID:       &discordUser.ID,
		DiscordUsername: &discordUser.Username,
		AvatarURL:       &avatarURL,
		AuthProvider:    strPtr("discord"),
		Role:            roles.User,
		Active:          false, // Manual activation by admin required
	}

	createdUser, err := h.userRepo.CreateDiscordUser(newUser)
	if err != nil {
		return nil, err
	}

	return createdUser, nil
}

// DiscordExchange handles frontend-initiated code exchange.
// Use this when the Discord redirect_uri points to the frontend (not the backend callback).
// The frontend receives the code from Discord, then calls this endpoint to exchange it for a JWT.
// Body: {"code": "...", "redirect_uri": "https://yourfrontend.com/auth/discord/callback"}
func (h *DiscordHandler) DiscordExchange(c *gin.Context) {
	var req struct {
		Code        string `json:"code" binding:"required"`
		RedirectURI string `json:"redirect_uri" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	token, err := h.oauth.ExchangeCodeWithURI(req.Code, req.RedirectURI)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to exchange Discord code", "details": err.Error()})
		return
	}

	discordUser, err := h.oauth.GetUser(token.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Discord user"})
		return
	}

	user, err := h.findOrCreateUser(discordUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process user"})
		return
	}

	if !user.Active {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is inactive. Contact the admin and login again."})
		return
	}

	jwtToken, err := GenerateJWT(strconv.Itoa(user.ID), string(user.Role), user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": jwtToken})
}

func (h *DiscordHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/auth/discord", h.DiscordLogin)
	router.GET("/auth/discord/callback", h.DiscordCallback)
	router.POST("/auth/discord/exchange", h.DiscordExchange)
}

func (h *DiscordHandler) RegisterProtectedRoutes(router *gin.RouterGroup) {
	router.POST("/users/:id/link-discord", Authorize("user"), h.LinkDiscord)
	router.POST("/users/:id/merge-discord", Authorize("moderator"), h.MergeDiscordAccount)
}

func (h *DiscordHandler) LinkDiscord(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if this is the account owner or admin
	if !IsOwnerOrAllowed(c, userID, "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data", "details": err.Error()})
		return
	}

	token, err := h.oauth.ExchangeCode(req.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Discord authorization error", "details": err.Error()})
		return
	}

	discordUser, err := h.oauth.GetUser(token.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch Discord data", "details": err.Error()})
		return
	}

	existingUser, _ := h.userRepo.FindUserByDiscordID(discordUser.ID)
	if existingUser != nil && existingUser.ID != userID {
		c.JSON(http.StatusConflict, gin.H{"error": "This Discord account is already linked to another user"})
		return
	}

	avatarURL := h.oauth.GetAvatarURL(discordUser)
	if err := h.userRepo.LinkDiscord(userID, discordUser.ID, discordUser.Username, avatarURL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to link accounts", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Discord account has been linked"})
}

func (h *DiscordHandler) MergeDiscordAccount(c *gin.Context) {
	targetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		SourceUserID int `json:"source_user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	if req.SourceUserID == targetID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Source and target user cannot be the same"})
		return
	}

	targetUser, err := h.userRepo.GetUser(targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch target user"})
		return
	}
	if targetUser == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Target user not found"})
		return
	}
	if targetUser.DiscordID != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Target user already has a Discord account linked"})
		return
	}

	sourceUser, err := h.userRepo.GetUser(req.SourceUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch source user"})
		return
	}
	if sourceUser == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Source user not found"})
		return
	}
	if sourceUser.DiscordID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Source user has no Discord account to transfer"})
		return
	}

	sourceDeleted, err := h.userRepo.MergeDiscordAccount(targetID, req.SourceUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to merge accounts", "details": err.Error()})
		return
	}

	message := "Discord account linked successfully. Source account was deactivated but could not be deleted due to existing data."
	if sourceDeleted {
		message = "Accounts merged successfully. Source account has been deleted."
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        message,
		"source_deleted": sourceDeleted,
	})
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func strPtr(s string) *string {
	return &s
}
