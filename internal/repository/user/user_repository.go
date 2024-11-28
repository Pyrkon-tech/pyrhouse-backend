package user

import (
	"fmt"
	"warehouse/internal/repository"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
)

type UserRepository struct {
	repository *repository.Repository
}

func (r *UserRepository) PersistUser(req models.UserRequest, hashedPassword []byte) error {
	query := r.repository.GoquDBWrapper.Insert("users").
		Rows(goqu.Record{
			"password_hash": string(hashedPassword),
			"username":      req.Username,
			"fullname":      req.Fullname,
			"role":          req.Role,
		})

	_, err := query.Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to insert User: %w", err)
	}

	return nil
}

func (r *UserRepository) GetUsers() ([]models.User, error) {
	var users []models.User
	query := r.repository.GoquDBWrapper.Select("id", "username", "fullname", "role").
		From("users")

	err := query.Executor().ScanStructs(&users)

	if err != nil {
		return nil, fmt.Errorf("error executing SQL statement: %w", err)
	}

	return users, nil
}

func NewRepository(r *repository.Repository) *UserRepository {
	return &UserRepository{repository: r}
}
