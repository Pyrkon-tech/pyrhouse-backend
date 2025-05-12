package security

import (
	"fmt"
	"log"
	"os"
	"time"
	"warehouse/internal/repository"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret []byte

func init() {
	log.Println("Inicjalizacja modułu security...")
	secret := os.Getenv("JWT_SECRET")
	log.Printf("Odczytana wartość JWT_SECRET: %v", secret != "")

	if secret == "" {
		log.Println("Próba ponownego załadowania zmiennych środowiskowych...")
		if err := godotenv.Load(); err != nil {
			log.Printf("Błąd ładowania .env: %v", err)
		}
		secret = os.Getenv("JWT_SECRET")
		log.Printf("Ponowna próba odczytu JWT_SECRET: %v", secret != "")
	}

	if secret == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	jwtSecret = []byte(secret)
	log.Println("Moduł security zainicjalizowany pomyślnie")
}

func AuthenticateUser(username, password string, repo *repository.Repository) (*models.User, error) {
	var user models.User

	query := repo.GoquDBWrapper.Select("id", "username", "password_hash", "role").From("users").Where(goqu.Ex{"username": username})

	if _, err := query.Executor().ScanStruct(&user); err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, err
	}

	return &user, nil
}

func GenerateJWT(userID string, role string, username string) (string, error) {
	claims := jwt.MapClaims{
		"userID":   userID,
		"role":     role,
		"username": username,
		"exp":      time.Now().Add(time.Hour * 120).Unix(), // 4 DAYS
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
