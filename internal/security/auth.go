package security

import (
	"fmt"
	"log"
	"strconv"
	"time"
	"warehouse/internal/config"
	"warehouse/internal/models"
	"warehouse/internal/repository"

	"github.com/doug-martin/goqu/v9"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Token audiences. The shop (shop.pyrhouse.space) signs with the same secret, so the
// warehouse must reject tokens issued for any other audience.
const (
	AudienceWarehouse = "pyrhouse-warehouse"
	AudienceShop      = "pyrhouse-shop"
)

var (
	jwtSecret         []byte
	jwtExpiration     time.Duration
	shopJWTExpiration time.Duration
)

func Initialize(cfg config.JWTConfig) error {
	if cfg.Secret == "" {
		return fmt.Errorf("JWT_SECRET is not configured")
	}

	jwtSecret = []byte(cfg.Secret)
	jwtExpiration = cfg.Expiration
	shopJWTExpiration = cfg.ShopExpiration
	if shopJWTExpiration <= 0 {
		shopJWTExpiration = 24 * time.Hour
	}

	log.Println("Security module initialized successfully")
	return nil
}

func AuthenticateUser(username, password string, repo *repository.Repository) (*models.User, error) {
	var user models.User

	query := repo.GoquDBWrapper.Select("id", "username", "password_hash", "role", "active").From("users").Where(goqu.Ex{"username": username})

	if _, err := query.Executor().ScanStruct(&user); err != nil {
		return nil, err
	}

	if !user.Active {
		return nil, fmt.Errorf("account is inactive")
	}

	// Check if the user has a password (Discord users may not have one)
	if user.PasswordHash == nil {
		return nil, fmt.Errorf("account does not have a password set - use Discord login")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return nil, err
	}

	return &user, nil
}

func GenerateJWT(userID string, role string, username string) (string, error) {
	claims := jwt.MapClaims{
		"userID":   userID,
		"role":     role,
		"username": username,
		"aud":      AudienceWarehouse,
		"exp":      time.Now().Add(jwtExpiration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// GenerateShopJWT is the only place that mints organizer shop tokens. sub is a shop_accounts.id,
// not a users.id, so the token must never be accepted by warehouse routes (aud keeps them apart).
func GenerateShopJWT(accountID int) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.Itoa(accountID),
		Audience:  jwt.ClaimStrings{AudienceShop},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(shopJWTExpiration)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
}

// ParseShopToken returns the shop account ID of a valid shop token. Warehouse tokens (aud
// pyrhouse-warehouse or no aud at all) are rejected.
func ParseShopToken(tokenString string) (int, error) {
	token, err := parseSignedToken(tokenString)
	if err != nil {
		return 0, err
	}
	if err := checkExclusiveAudience(token, AudienceShop, false); err != nil {
		return 0, err
	}
	sub, err := token.Claims.GetSubject()
	if err != nil || sub == "" {
		return 0, fmt.Errorf("token has no subject")
	}
	accountID, err := strconv.Atoi(sub)
	if err != nil || accountID <= 0 {
		return 0, fmt.Errorf("invalid token subject")
	}
	return accountID, nil
}

func GetUserIDFromToken(c *gin.Context) (string, error) {
	token, err := getTokenFromContext(c)

	if err != nil {
		return "", err
	}

	claims := token.Claims.(jwt.MapClaims)
	userID, ok := claims["userID"].(string)
	if !ok {
		return "", fmt.Errorf("userID is not a string")
	}

	return userID, nil
}
