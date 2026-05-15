package reservations

import (
	"fmt"
	"time"
	"warehouse/internal/repository"

	"github.com/doug-martin/goqu/v9"
	"github.com/lib/pq"
)

type Repository struct {
	repo *repository.Repository
}

func NewRepository(r *repository.Repository) *Repository {
	return &Repository{repo: r}
}

func (r *Repository) CreateReservations(pyrCodes []string, categoryID int) ([]PyrCodeReservation, error) {
	rows := make([]goqu.Record, len(pyrCodes))
	for i, code := range pyrCodes {
		rows[i] = goqu.Record{
			"pyr_code":    code,
			"category_id": categoryID,
		}
	}

	var created []PyrCodeReservation
	query := r.repo.GoquDBWrapper.
		Insert("pyr_code_reservations").
		Rows(rows).
		Returning("id", "pyr_code", "category_id", "reserved_at")

	if err := query.Executor().ScanStructs(&created); err != nil {
		return nil, fmt.Errorf("failed to create reservations: %w", err)
	}

	return created, nil
}

func (r *Repository) GetReservations(categoryID *int, status string) ([]PyrCodeReservation, error) {
	query := r.repo.GoquDBWrapper.
		Select("id", "pyr_code", "category_id", "reserved_at", "claimed_at", "item_id").
		From("pyr_code_reservations").
		Order(goqu.I("id").Asc())

	switch status {
	case "free":
		query = query.Where(goqu.I("claimed_at").IsNull())
	case "claimed":
		query = query.Where(goqu.I("claimed_at").IsNotNull())
	}

	if categoryID != nil {
		query = query.Where(goqu.Ex{"category_id": *categoryID})
	}

	var result []PyrCodeReservation
	if err := query.Executor().ScanStructs(&result); err != nil {
		return nil, fmt.Errorf("failed to get reservations: %w", err)
	}

	return result, nil
}

// FindUnclaimedByPyrCodes fetches and locks rows for claim (must be called inside a transaction).
func (r *Repository) FindUnclaimedByPyrCodes(tx *goqu.TxDatabase, pyrCodes []string) ([]PyrCodeReservation, error) {
	sql := `
		SELECT id, pyr_code, category_id, reserved_at, claimed_at, item_id
		FROM pyr_code_reservations
		WHERE pyr_code = ANY($1) AND claimed_at IS NULL
		FOR UPDATE
	`
	rows, err := tx.Query(sql, pq.Array(pyrCodes))
	if err != nil {
		return nil, fmt.Errorf("failed to lock reservation rows: %w", err)
	}
	defer rows.Close()

	var result []PyrCodeReservation
	for rows.Next() {
		var res PyrCodeReservation
		if err := rows.Scan(&res.ID, &res.PyrCode, &res.CategoryID, &res.ReservedAt, &res.ClaimedAt, &res.ItemID); err != nil {
			return nil, fmt.Errorf("failed to scan reservation: %w", err)
		}
		result = append(result, res)
	}
	return result, rows.Err()
}

func (r *Repository) MarkClaimed(tx *goqu.TxDatabase, reservationID, itemID int) error {
	now := time.Now()
	result, err := tx.Update("pyr_code_reservations").
		Set(goqu.Record{"claimed_at": now, "item_id": itemID}).
		Where(goqu.Ex{"id": reservationID}).
		Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to mark reservation claimed: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("reservation %d not found", reservationID)
	}
	return nil
}

// DeleteByPyrCodes removes unclaimed reservations by pyr_code.
// Returns claimed pyr_codes if any are already claimed (caller should 409).
func (r *Repository) DeleteByPyrCodes(pyrCodes []string) (int, []string, error) {
	claimed, err := r.findClaimedAmong("pyr_code", pyrCodes)
	if err != nil {
		return 0, nil, err
	}
	if len(claimed) > 0 {
		return 0, claimed, nil
	}

	result, err := r.repo.GoquDBWrapper.
		Delete("pyr_code_reservations").
		Where(goqu.Ex{"pyr_code": pyrCodes, "claimed_at": nil}).
		Executor().Exec()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to delete reservations: %w", err)
	}
	n, _ := result.RowsAffected()
	return int(n), nil, nil
}

// DeleteByIDs removes unclaimed reservations by id.
func (r *Repository) DeleteByIDs(ids []int) (int, []string, error) {
	claimed, err := r.findClaimedAmong("id", ids)
	if err != nil {
		return 0, nil, err
	}
	if len(claimed) > 0 {
		return 0, claimed, nil
	}

	result, err := r.repo.GoquDBWrapper.
		Delete("pyr_code_reservations").
		Where(goqu.Ex{"id": ids, "claimed_at": nil}).
		Executor().Exec()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to delete reservations: %w", err)
	}
	n, _ := result.RowsAffected()
	return int(n), nil, nil
}

func (r *Repository) findClaimedAmong(col string, values interface{}) ([]string, error) {
	rows, err := r.repo.GoquDBWrapper.
		Select("pyr_code").
		From("pyr_code_reservations").
		Where(goqu.Ex{col: values}).
		Where(goqu.I("claimed_at").IsNotNull()).
		Executor().Query()
	if err != nil {
		return nil, fmt.Errorf("failed to check claimed reservations: %w", err)
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}
