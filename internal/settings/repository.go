package settings

import (
	"context"
	"time"

	"warehouse/internal/repository"

	"github.com/doug-martin/goqu/v9"
)

type Repository struct {
	repo *repository.Repository
}

func NewRepository(repo *repository.Repository) *Repository {
	return &Repository{repo: repo}
}

func (r *Repository) Get(ctx context.Context, key string) (string, error) {
	var value string
	_, err := r.repo.GoquDBWrapper.Select("value").
		From("app_settings").
		Where(goqu.C("key").Eq(key)).
		Executor().ScanVal(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (r *Repository) GetSetting(key string) (*AppSettings, error) {
	var setting AppSettings
	found, err := r.repo.GoquDBWrapper.Select(
		"key", "value", "description", "updated_at",
	).
		From("app_settings").
		Where(goqu.C("key").Eq(key)).
		Executor().ScanStruct(&setting)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &setting, nil
}

func (r *Repository) GetAll() ([]AppSettings, error) {
	var settings []AppSettings
	err := r.repo.GoquDBWrapper.Select(
		"key", "description", "updated_at",
	).
		From("app_settings").
		Order(goqu.C("key").Asc()).
		Executor().ScanStructs(&settings)
	if err != nil {
		return nil, err
	}
	return settings, nil
}

func (r *Repository) GetByPrefix(prefix string) ([]AppSettings, error) {
	var settings []AppSettings
	err := r.repo.GoquDBWrapper.Select(
		"key", "value", "description", "updated_at",
	).
		From("app_settings").
		Where(goqu.C("key").Like(prefix+"%")).
		Order(goqu.C("key").Asc()).
		Executor().ScanStructs(&settings)
	if err != nil {
		return nil, err
	}
	return settings, nil
}

func (r *Repository) UpdateSetting(key string, value string) error {
	now := time.Now()
	_, err := r.repo.GoquDBWrapper.Update("app_settings").
		Set(goqu.Record{
			"value":      value,
			"updated_at": now,
		}).
		Where(goqu.C("key").Eq(key)).
		Executor().Exec()
	return err
}
