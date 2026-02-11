package security

import (
	"fmt"
	"log"
	"time"
	"warehouse/internal/config"
	"warehouse/internal/models"
	"warehouse/internal/repository"

	"github.com/doug-martin/goqu/v9"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	jwtSecret     []byte
	jwtExpiration time.Duration
)

func Initialize(cfg config.JWTConfig) error {
	if cfg.Secret == "" {
		return fmt.Errorf("JWT_SECRET is not configured")
	}

	jwtSecret = []byte(cfg.Secret)
	jwtExpiration = cfg.Expiration

	log.Println("Moduł security zainicjalizowany pomyślnie")
	return nil
}

func AuthenticateUser(username, password string, repo *repository.Repository) (*models.User, error) {
	var user models.User

	query := repo.GoquDBWrapper.Select("id", "username", "password_hash", "role", "active").From("users").Where(goqu.Ex{"username": username})

	if _, err := query.Executor().ScanStruct(&user); err != nil {
		return nil, err
	}

	if !user.Active {
		return nil, fmt.Errorf("konto jest nieaktywne")
	}

	// Sprawdź czy użytkownik ma hasło (użytkownicy Discord mogą nie mieć)
	if user.PasswordHash == nil {
		return nil, fmt.Errorf("konto nie ma ustawionego hasła - użyj logowania przez Discord")
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
		"exp":      time.Now().Add(jwtExpiration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
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
