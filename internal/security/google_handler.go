package security

import (
	"net/http"
	"strconv"
	"strings"
	"warehouse/internal/models"
	"warehouse/internal/oauth"
	"warehouse/internal/roles"

	"github.com/gin-gonic/gin"
)

const allowedDomain = "@pyrkon.pl"

var allowedEmails = map[string]bool{
	"warrmag7@gmail.com": true,
}

type GoogleUserRepository interface {
	FindUserByGoogleID(googleID string) (*models.User, error)
	FindUserByUsername(username string) (*models.User, error)
	CreateGoogleUser(user *models.User) (*models.User, error)
	UpdateGoogleInfo(userID int, avatarURL string) error
	LinkGoogle(userID int, googleID, googleEmail, avatarURL string) error
}

type GoogleHandler struct {
	oauth    *oauth.GoogleOAuth
	userRepo GoogleUserRepository
}

func NewGoogleHandler(oauth *oauth.GoogleOAuth, userRepo GoogleUserRepository) *GoogleHandler {
	return &GoogleHandler{oauth: oauth, userRepo: userRepo}
}

func isAllowedGoogleEmail(email string) bool {
	return strings.HasSuffix(email, allowedDomain) || allowedEmails[email]
}

func (h *GoogleHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/auth/google", h.GoogleLogin)
	router.POST("/auth/google/exchange", h.GoogleExchange)
}

func (h *GoogleHandler) RegisterProtectedRoutes(router *gin.RouterGroup) {
	router.POST("/users/:id/link-google", Authorize("user"), h.LinkGoogle)
}

func (h *GoogleHandler) GoogleLogin(c *gin.Context) {
	state := generateState()

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

// GoogleExchange handles frontend-initiated code exchange.
// Frontend receives the code from Google callback, then calls this endpoint to exchange it for a JWT.
// Body: {"code": "...", "redirect_uri": "https://yourfrontend.com/auth/google/callback"}
func (h *GoogleHandler) GoogleExchange(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to exchange Google code", "details": err.Error()})
		return
	}

	googleUser, err := h.oauth.GetUser(token.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Google user info"})
		return
	}

	if !isAllowedGoogleEmail(googleUser.Email) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access restricted to " + allowedDomain + " accounts"})
		return
	}

	user, err := h.findOrCreateGoogleUser(googleUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process user"})
		return
	}

	if !user.Active {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is inactive. Contact the admin."})
		return
	}

	jwtToken, err := GenerateJWT(strconv.Itoa(user.ID), string(user.Role), user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": jwtToken})
}

func (h *GoogleHandler) LinkGoogle(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if !IsOwnerOrAllowed(c, userID, "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to exchange Google code", "details": err.Error()})
		return
	}

	googleUser, err := h.oauth.GetUser(token.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Google user info"})
		return
	}

	if !isAllowedGoogleEmail(googleUser.Email) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access restricted to " + allowedDomain + " accounts"})
		return
	}

	existing, err := h.userRepo.FindUserByGoogleID(googleUser.Sub)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check Google account"})
		return
	}
	if existing != nil && existing.ID != userID {
		c.JSON(http.StatusConflict, gin.H{"error": "This Google account is already linked to another user"})
		return
	}

	if err := h.userRepo.LinkGoogle(userID, googleUser.Sub, googleUser.Email, googleUser.Picture); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to link Google account", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Google account linked successfully"})
}

func (h *GoogleHandler) findOrCreateGoogleUser(googleUser *oauth.GoogleUser) (*models.User, error) {
	user, err := h.userRepo.FindUserByGoogleID(googleUser.Sub)
	if err != nil {
		return nil, err
	}

	if user != nil {
		if googleUser.Picture != "" {
			_ = h.userRepo.UpdateGoogleInfo(user.ID, googleUser.Picture)
		}
		return user, nil
	}

	// derive username from email prefix (e.g. jan.kowalski from jan.kowalski@pyrkon.pl)
	username := strings.SplitN(googleUser.Email, "@", 2)[0]

	// if an account with this username already exists and has no Google linked — auto-link (same as Discord)
	existing, err := h.userRepo.FindUserByUsername(username)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.GoogleID == nil {
		if err := h.userRepo.LinkGoogle(existing.ID, googleUser.Sub, googleUser.Email, googleUser.Picture); err != nil {
			return nil, err
		}
		existing.GoogleID = &googleUser.Sub
		existing.GoogleEmail = &googleUser.Email
		return existing, nil
	}

	picture := googleUser.Picture
	provider := "google"

	newUser := &models.User{
		Username:     username,
		GoogleID:     &googleUser.Sub,
		GoogleEmail:  &googleUser.Email,
		AvatarURL:    &picture,
		AuthProvider: &provider,
		Role:         roles.User,
		Active:       true, // @pyrkon.pl is a trusted domain — auto-activate
	}

	return h.userRepo.CreateGoogleUser(newUser)
}
