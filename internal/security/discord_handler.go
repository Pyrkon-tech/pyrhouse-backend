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

// DiscordUserRepository definiuje metody repozytorium wymagane przez DiscordHandler
type DiscordUserRepository interface {
	FindUserByDiscordID(discordID string) (*models.User, error)
	FindUserByUsername(username string) (*models.User, error)
	CreateDiscordUser(user *models.User) (*models.User, error)
	UpdateDiscordInfo(userID int, username string, avatarURL string) error
	LinkDiscord(userID int, discordID, discordUsername, avatarURL string) error
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

// DiscordLogin redirectuje do strony autoryzacji Discord
func (h *DiscordHandler) DiscordLogin(c *gin.Context) {
	state := generateState()
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)
	c.Redirect(http.StatusTemporaryRedirect, h.oauth.GetAuthURL(state))
}

// DiscordCallback obsługuje callback z Discord
func (h *DiscordHandler) DiscordCallback(c *gin.Context) {
	frontendURL := h.oauth.GetFrontendURL()

	// Helper do przekierowania z błędem
	redirectWithError := func(errorMsg string) {
		if frontendURL != "" {
			redirectURL := frontendURL + "?error=" + url.QueryEscape(errorMsg)
			c.Redirect(http.StatusTemporaryRedirect, redirectURL)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": errorMsg})
		}
	}

	// 1. Weryfikacja state (CSRF protection)
	state := c.Query("state")
	savedState, _ := c.Cookie("oauth_state")
	if state != savedState {
		redirectWithError("Invalid state parameter")
		return
	}

	// 2. Pobranie code
	code := c.Query("code")
	if code == "" {
		redirectWithError("Missing authorization code")
		return
	}

	// 3. Wymiana code na token
	token, err := h.oauth.ExchangeCode(code)
	if err != nil {
		redirectWithError("Failed to exchange code")
		return
	}

	// 4. Pobranie danych użytkownika z Discord
	discordUser, err := h.oauth.GetUser(token.AccessToken)
	if err != nil {
		redirectWithError("Failed to get Discord user")
		return
	}

	// 5. Znalezienie lub utworzenie użytkownika
	user, err := h.findOrCreateUser(discordUser)
	if err != nil {
		redirectWithError("Failed to process user")
		return
	}

	// 6. Sprawdzenie czy konto aktywne
	if !user.Active {
		redirectWithError("Konto jest nieaktywne. Skontaktuj się z administratorem.")
		return
	}

	// 7. Generowanie JWT
	jwtToken, err := GenerateJWT(strconv.Itoa(user.ID), string(user.Role), user.Username)
	if err != nil {
		redirectWithError("Failed to generate token")
		return
	}

	// 8. Przekierowanie na frontend z tokenem
	if frontendURL != "" {
		redirectURL := frontendURL + "?token=" + jwtToken
		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
	} else {
		// Fallback - zwróć JSON jeśli brak frontendURL
		c.JSON(http.StatusOK, gin.H{"token": jwtToken})
	}
}

func (h *DiscordHandler) findOrCreateUser(discordUser *oauth.DiscordUser) (*models.User, error) {
	// Szukaj użytkownika po discord_id
	user, err := h.userRepo.FindUserByDiscordID(discordUser.ID)
	if err != nil {
		return nil, err
	}

	if user != nil {
		// Użytkownik istnieje - aktualizuj dane Discord (username, avatar mogły się zmienić)
		avatarURL := h.oauth.GetAvatarURL(discordUser)
		if err := h.userRepo.UpdateDiscordInfo(user.ID, discordUser.Username, avatarURL); err != nil {
			// Logujemy błąd, ale kontynuujemy
		}
		return user, nil
	}

	// Sprawdź czy istnieje konto z taką samą nazwą użytkownika (bez połączenia z Discord)
	existingUser, err := h.userRepo.FindUserByUsername(discordUser.Username)
	if err != nil {
		return nil, err
	}
	if existingUser != nil && existingUser.DiscordID == nil {
		// Istnieje konto z taką nazwą - automatycznie połącz z Discord
		avatarURL := h.oauth.GetAvatarURL(discordUser)
		if err := h.userRepo.LinkDiscord(existingUser.ID, discordUser.ID, discordUser.Username, avatarURL); err != nil {
			return nil, err
		}
		// Pobierz zaktualizowanego użytkownika
		existingUser.DiscordID = &discordUser.ID
		existingUser.DiscordUsername = &discordUser.Username
		existingUser.AvatarURL = &avatarURL
		return existingUser, nil
	}

	// Utwórz nowego użytkownika
	avatarURL := h.oauth.GetAvatarURL(discordUser)
	newUser := &models.User{
		Username:        discordUser.Username,
		DiscordID:       &discordUser.ID,
		DiscordUsername: &discordUser.Username,
		AvatarURL:       &avatarURL,
		AuthProvider:    "discord",
		Role:            roles.User,
		Active:          false, // Wymagana ręczna aktywacja przez admina
	}

	createdUser, err := h.userRepo.CreateDiscordUser(newUser)
	if err != nil {
		return nil, err
	}

	return createdUser, nil
}

// RegisterRoutes rejestruje endpointy Discord OAuth
func (h *DiscordHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/auth/discord", h.DiscordLogin)
	router.GET("/auth/discord/callback", h.DiscordCallback)
}

// RegisterProtectedRoutes rejestruje chronione endpointy Discord
func (h *DiscordHandler) RegisterProtectedRoutes(router *gin.RouterGroup) {
	router.POST("/users/:id/link-discord", Authorize("user"), h.LinkDiscord)
}

// LinkDiscord łączy istniejące konto z Discord
func (h *DiscordHandler) LinkDiscord(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowe ID użytkownika"})
		return
	}

	// Sprawdź czy to właściciel konta lub admin
	if !IsOwnerOrAllowed(c, userID, "admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Brak uprawnień"})
		return
	}

	var req struct {
		Code  string `json:"code" binding:"required"`
		State string `json:"state" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowe dane", "details": err.Error()})
		return
	}

	// Wymiana code na token
	token, err := h.oauth.ExchangeCode(req.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Błąd autoryzacji Discord", "details": err.Error()})
		return
	}

	// Pobranie danych Discord
	discordUser, err := h.oauth.GetUser(token.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nie udało się pobrać danych Discord", "details": err.Error()})
		return
	}

	// Sprawdź czy discord_id nie jest już przypisany do innego konta
	existingUser, _ := h.userRepo.FindUserByDiscordID(discordUser.ID)
	if existingUser != nil && existingUser.ID != userID {
		c.JSON(http.StatusConflict, gin.H{"error": "To konto Discord jest już połączone z innym użytkownikiem"})
		return
	}

	// Połącz konta
	avatarURL := h.oauth.GetAvatarURL(discordUser)
	if err := h.userRepo.LinkDiscord(userID, discordUser.ID, discordUser.Username, avatarURL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nie udało się połączyć kont", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Konto Discord zostało połączone"})
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
