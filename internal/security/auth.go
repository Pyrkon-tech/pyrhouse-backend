package security

import (
	"database/sql"
	"time"
	"warehouse/pkg/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("your_secret_key")

// AuthenticateUser verifies username and password from the database.
func AuthenticateUser(username, password string, db *sql.DB) (*models.User, error) {
	var user models.User

	row := db.QueryRow("SELECT id, username, password_hash, role FROM users WHERE username = $1", username)
	if err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role); err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, err
	}

	return &user, nil
}

// GenerateJWT generates a new JWT for a user.
func GenerateJWT(userID string, role string) (string, error) {
	claims := jwt.MapClaims{
		"userID": userID,
		"role":   role,
		"exp":    time.Now().Add(time.Hour * 1).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
