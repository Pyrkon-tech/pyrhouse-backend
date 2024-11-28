package security

import (
	"time"
	"warehouse/internal/repository"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// TODO CHANGE ME FFS
var jwtSecret = []byte("your_secret_key")

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

func GenerateJWT(userID string, role string) (string, error) {
	claims := jwt.MapClaims{
		"userID": userID,
		"role":   role,
		"exp":    time.Now().Add(time.Hour * 120).Unix(), // 4 DAYS
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
