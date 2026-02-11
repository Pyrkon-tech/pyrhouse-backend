package users

import (
	"fmt"
	"time"
	"warehouse/internal/models"
	"warehouse/internal/repository"

	"github.com/doug-martin/goqu/v9"
	"github.com/lib/pq"
)

type UserRepository interface {
	PersistUser(req models.CreateUserRequest, hashedPassword []byte) error
	GetUser(id int) (*models.User, error)
	IsUsernameUnique(username string) (bool, error)
	GetUsers() ([]models.User, error)
	AddUserPoints(id int, points int) error
	UpdateUser(id int, changes *models.UserChanges) error
	DeleteUser(id int) error
	UsersExists(userIDs []int) (bool, error)
	// Discord OAuth methods
	FindUserByDiscordID(discordID string) (*models.User, error)
	FindUserByUsername(username string) (*models.User, error)
	CreateDiscordUser(user *models.User) (*models.User, error)
	UpdateDiscordInfo(userID int, username string, avatarURL string) error
	LinkDiscord(userID int, discordID, discordUsername, avatarURL string) error
}

type userRepositoryImpl struct {
	repository *repository.Repository
}

func (r *userRepositoryImpl) PersistUser(req models.CreateUserRequest, hashedPassword []byte) error {
	query := r.repository.GoquDBWrapper.Insert("users").
		Rows(goqu.Record{
			"password_hash": string(hashedPassword),
			"username":      req.Username,
			"fullname":      req.Fullname,
			"role":          req.Role,
			"points":        req.Points,
			"active":        req.Active,
		})

	_, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to insert User: %w", err)
	}

	return nil
}

func (r *userRepositoryImpl) GetUsers() ([]models.User, error) {
	var users []models.User
	query := r.repository.GoquDBWrapper.Select("id", "username", "fullname", "role", "points", "active").
		From("users")

	err := query.Executor().ScanStructs(&users)

	if err != nil {
		return nil, fmt.Errorf("error executing SQL statement: %w", err)
	}

	return users, nil
}

func (r *userRepositoryImpl) GetUser(id int) (*models.User, error) {
	var user models.User
	query := r.repository.GoquDBWrapper.Select("id", "username", "fullname", "password_hash", "role", "points", "active").
		From("users").
		Where(goqu.Ex{"id": id})

	_, err := query.Executor().ScanStruct(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (r *userRepositoryImpl) SetUserActive(userID int, active bool) error {
	query := r.repository.GoquDBWrapper.Update("users").
		Set(goqu.Record{"active": active}).
		Where(goqu.Ex{"id": userID})

	_, err := query.Executor().Exec()
	return err
}

func (r *userRepositoryImpl) IsUsernameUnique(username string) (bool, error) {
	var count int

	query := `SELECT COUNT(*) FROM users WHERE username = $1`
	err := r.repository.DB.QueryRow(query, username).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to get user: %w", err)
	}

	return count == 0, nil
}

func (r *userRepositoryImpl) AddUserPoints(id int, points int) error {
	query := r.repository.GoquDBWrapper.Update("users").
		Set(goqu.Record{"points": goqu.L("points + ?", points)}).
		Where(goqu.Ex{"id": id})

	_, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to add user points: %w", err)
	}

	return nil
}

func (r *userRepositoryImpl) UpdateUser(id int, changes *models.UserChanges) error {
	updateFields := make(goqu.Record)

	if changes.PasswordHash != nil {
		updateFields["password_hash"] = *changes.PasswordHash
	}
	if changes.Role != nil {
		updateFields["role"] = *changes.Role
	}
	if changes.Points != nil {
		updateFields["points"] = *changes.Points
	}

	if changes.Fullname != nil {
		updateFields["fullname"] = *changes.Fullname
	}

	if changes.Username != nil {
		updateFields["username"] = *changes.Username
	}

	if changes.Active != nil {
		updateFields["active"] = *changes.Active
	}

	query := r.repository.GoquDBWrapper.Update("users").
		Set(updateFields).
		Where(goqu.Ex{"id": id})

	_, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (r *userRepositoryImpl) DeleteUser(id int) error {
	result, err := r.repository.GoquDBWrapper.Delete("users").Where(goqu.Ex{"id": id}).Executor().Exec()
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return fmt.Errorf("nie można usunąć użytkownika, ponieważ ma przypisane transfery")
		}
		return fmt.Errorf("błąd podczas usuwania użytkownika: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("nie można sprawdzić liczby usuniętych wierszy: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("nie znaleziono użytkownika o id: %d", id)
	}

	return nil
}

func NewRepository(r *repository.Repository) UserRepository {
	return &userRepositoryImpl{repository: r}
}

func (r *userRepositoryImpl) UsersExists(userIDs []int) (bool, error) {
	type UserID struct {
		ID int `db:"id"`
	}
	var dbUserIDs []UserID
	query := r.repository.GoquDBWrapper.Select("id").From("users").Where(goqu.Ex{"id": userIDs})

	err := query.Executor().ScanStructs(&dbUserIDs)
	if err != nil {
		return false, fmt.Errorf("failed to get users: %w", err)
	}

	return len(dbUserIDs) == len(userIDs), nil
}

// FindUserByDiscordID znajduje użytkownika po jego Discord ID
func (r *userRepositoryImpl) FindUserByDiscordID(discordID string) (*models.User, error) {
	var user models.User
	query := r.repository.GoquDBWrapper.Select(
		"id", "username", "fullname", "password_hash", "role", "points", "active",
		"discord_id", "discord_username", "avatar_url", "auth_provider",
	).From("users").Where(goqu.Ex{"discord_id": discordID})

	found, err := query.Executor().ScanStruct(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by discord_id: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &user, nil
}

// FindUserByUsername znajduje użytkownika po nazwie użytkownika
func (r *userRepositoryImpl) FindUserByUsername(username string) (*models.User, error) {
	var user models.User
	query := r.repository.GoquDBWrapper.Select(
		"id", "username", "fullname", "password_hash", "role", "points", "active",
		"discord_id", "discord_username", "avatar_url", "auth_provider",
	).From("users").Where(goqu.Ex{"username": username})

	found, err := query.Executor().ScanStruct(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by username: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &user, nil
}

// CreateDiscordUser tworzy nowego użytkownika z danymi Discord
func (r *userRepositoryImpl) CreateDiscordUser(user *models.User) (*models.User, error) {
	// Sprawdź czy username jest unikalna, jeśli nie - wygeneruj unikalną
	username := user.Username
	isUnique, err := r.IsUsernameUnique(username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username uniqueness: %w", err)
	}

	if !isUnique {
		// Username zajęta - dodaj Discord ID jako sufiks
		if user.DiscordID != nil {
			username = fmt.Sprintf("%s_%s", user.Username, *user.DiscordID)
		} else {
			// Fallback - dodaj losowy sufiks
			username = fmt.Sprintf("%s_%d", user.Username, r.generateRandomSuffix())
		}
	}

	query := r.repository.GoquDBWrapper.Insert("users").
		Rows(goqu.Record{
			"username":         username,
			"discord_id":       user.DiscordID,
			"discord_username": user.DiscordUsername,
			"avatar_url":       user.AvatarURL,
			"auth_provider":    user.AuthProvider,
			"role":             user.Role,
			"active":           user.Active,
		}).
		Returning("id")

	var id int
	_, err = query.Executor().ScanVal(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create discord user: %w", err)
	}

	user.ID = id
	user.Username = username
	return user, nil
}

func (r *userRepositoryImpl) generateRandomSuffix() int64 {
	return time.Now().UnixNano() % 100000
}

// UpdateDiscordInfo aktualizuje dane Discord użytkownika
func (r *userRepositoryImpl) UpdateDiscordInfo(userID int, username string, avatarURL string) error {
	query := r.repository.GoquDBWrapper.Update("users").
		Set(goqu.Record{
			"discord_username": username,
			"avatar_url":       avatarURL,
		}).
		Where(goqu.Ex{"id": userID})

	_, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to update discord info: %w", err)
	}
	return nil
}

// LinkDiscord łączy istniejące konto z kontem Discord
func (r *userRepositoryImpl) LinkDiscord(userID int, discordID, discordUsername, avatarURL string) error {
	query := r.repository.GoquDBWrapper.Update("users").
		Set(goqu.Record{
			"discord_id":       discordID,
			"discord_username": discordUsername,
			"avatar_url":       avatarURL,
		}).
		Where(goqu.Ex{"id": userID})

	_, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to link discord: %w", err)
	}
	return nil
}
