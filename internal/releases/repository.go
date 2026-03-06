package releases

import (
	"fmt"
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

// SuggestAssets returns assets matching the given origin and optional location, excluding in_transit.
func (r *Repository) SuggestAssets(originID int, locationID *int) ([]SuggestedAsset, error) {
	query := r.repo.GoquDBWrapper.Select(
		goqu.I("i.id"),
		goqu.I("i.pyr_code"),
		goqu.I("i.item_serial"),
		goqu.I("i.status"),
		goqu.I("c.label").As("category_name"),
		goqu.L("CASE WHEN i.origin_suffix IS NOT NULL THEN o.slug || '-' || i.origin_suffix ELSE o.slug END").As("origin_label"),
		goqu.I("l.name").As("location_name"),
	).
		From(goqu.T("items").As("i")).
		LeftJoin(goqu.T("origins").As("o"), goqu.On(goqu.Ex{"i.origin_id": goqu.I("o.id")})).
		LeftJoin(goqu.T("item_category").As("c"), goqu.On(goqu.Ex{"i.item_category_id": goqu.I("c.id")})).
		LeftJoin(goqu.T("locations").As("l"), goqu.On(goqu.Ex{"i.location_id": goqu.I("l.id")})).
		Where(
			goqu.Ex{"i.origin_id": originID},
			goqu.I("i.status").In([]string{"available", "located"}),
		).
		Order(goqu.I("i.id").Asc())

	if locationID != nil {
		query = query.Where(goqu.Ex{"i.location_id": *locationID})
	}

	var assets []SuggestedAsset
	if err := query.Executor().ScanStructs(&assets); err != nil {
		return nil, fmt.Errorf("failed to suggest assets: %w", err)
	}
	if assets == nil {
		assets = []SuggestedAsset{}
	}
	return assets, nil
}

// SuggestStocks returns stocks matching the given origin and optional location.
func (r *Repository) SuggestStocks(originID int, locationID *int) ([]SuggestedStock, error) {
	query := r.repo.GoquDBWrapper.Select(
		goqu.I("s.id"),
		goqu.I("s.quantity"),
		goqu.I("c.label").As("category_name"),
		goqu.L("CASE WHEN s.origin_suffix IS NOT NULL THEN o.slug || '-' || s.origin_suffix ELSE o.slug END").As("origin_label"),
		goqu.I("l.name").As("location_name"),
	).
		From(goqu.T("non_serialized_items").As("s")).
		LeftJoin(goqu.T("origins").As("o"), goqu.On(goqu.Ex{"s.origin_id": goqu.I("o.id")})).
		LeftJoin(goqu.T("item_category").As("c"), goqu.On(goqu.Ex{"s.item_category_id": goqu.I("c.id")})).
		LeftJoin(goqu.T("locations").As("l"), goqu.On(goqu.Ex{"s.location_id": goqu.I("l.id")})).
		Where(
			goqu.Ex{"s.origin_id": originID},
			goqu.I("s.quantity").Gt(0),
		).
		Order(goqu.I("s.id").Asc())

	if locationID != nil {
		query = query.Where(goqu.Ex{"s.location_id": *locationID})
	}

	var stocks []SuggestedStock
	if err := query.Executor().ScanStructs(&stocks); err != nil {
		return nil, fmt.Errorf("failed to suggest stocks: %w", err)
	}
	if stocks == nil {
		stocks = []SuggestedStock{}
	}
	return stocks, nil
}

// CreateRelease inserts the release header and returns the created release.
func (r *Repository) CreateRelease(tx *goqu.TxDatabase, ref string, originID int, notes *string, createdBy int) (*Release, error) {
	record := goqu.Record{
		"reference":  ref,
		"origin_id":  originID,
		"status":     "draft",
		"created_by": createdBy,
	}
	if notes != nil {
		record["notes"] = *notes
	}

	var release Release
	_, err := tx.Insert("releases").Rows(record).Returning("id", "reference", "origin_id", "notes", "status", "created_by", "completed_at", "created_at").Executor().ScanStruct(&release)
	if err != nil {
		return nil, fmt.Errorf("failed to create release: %w", err)
	}
	return &release, nil
}

// InsertReleaseAssets inserts asset snapshot records for a release.
func (r *Repository) InsertReleaseAssets(tx *goqu.TxDatabase, releaseID int, itemIDs []int) error {
	if len(itemIDs) == 0 {
		return nil
	}

	// Fetch snapshot data from live items
	query := r.repo.GoquDBWrapper.Select(
		goqu.I("i.id").As("item_id"),
		goqu.I("i.pyr_code"),
		goqu.I("i.item_serial"),
		goqu.I("c.label").As("category_name"),
		goqu.L("CASE WHEN i.origin_suffix IS NOT NULL THEN o.slug || '-' || i.origin_suffix ELSE o.slug END").As("origin_label"),
		goqu.I("l.name").As("location_name"),
	).
		From(goqu.T("items").As("i")).
		LeftJoin(goqu.T("origins").As("o"), goqu.On(goqu.Ex{"i.origin_id": goqu.I("o.id")})).
		LeftJoin(goqu.T("item_category").As("c"), goqu.On(goqu.Ex{"i.item_category_id": goqu.I("c.id")})).
		LeftJoin(goqu.T("locations").As("l"), goqu.On(goqu.Ex{"i.location_id": goqu.I("l.id")})).
		Where(goqu.I("i.id").In(itemIDs))

	type assetSnapshot struct {
		ItemID       int     `db:"item_id"`
		PyrCode      *string `db:"pyr_code"`
		ItemSerial   *string `db:"item_serial"`
		CategoryName *string `db:"category_name"`
		OriginLabel  *string `db:"origin_label"`
		LocationName *string `db:"location_name"`
	}

	var snapshots []assetSnapshot
	if err := query.Executor().ScanStructs(&snapshots); err != nil {
		return fmt.Errorf("failed to fetch asset snapshots: %w", err)
	}

	if len(snapshots) != len(itemIDs) {
		return fmt.Errorf("some assets not found: expected %d, found %d", len(itemIDs), len(snapshots))
	}

	rows := make([]interface{}, len(snapshots))
	for i, s := range snapshots {
		rows[i] = goqu.Record{
			"release_id":    releaseID,
			"item_id":       s.ItemID,
			"pyr_code":      s.PyrCode,
			"item_serial":   s.ItemSerial,
			"category_name": s.CategoryName,
			"origin_label":  s.OriginLabel,
			"location_name": s.LocationName,
		}
	}

	_, err := tx.Insert("release_assets").Rows(rows...).Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to insert release assets: %w", err)
	}
	return nil
}

// InsertReleaseStocks inserts stock snapshot records for a release.
func (r *Repository) InsertReleaseStocks(tx *goqu.TxDatabase, releaseID int, stocks []StockReleaseReq) error {
	if len(stocks) == 0 {
		return nil
	}

	stockIDs := make([]int, len(stocks))
	qtyMap := make(map[int]int)
	for i, s := range stocks {
		stockIDs[i] = s.StockID
		qtyMap[s.StockID] = s.Quantity
	}

	query := r.repo.GoquDBWrapper.Select(
		goqu.I("s.id").As("stock_id"),
		goqu.I("s.item_category_id"),
		goqu.I("s.quantity"),
		goqu.I("c.label").As("category_name"),
		goqu.L("CASE WHEN s.origin_suffix IS NOT NULL THEN o.slug || '-' || s.origin_suffix ELSE o.slug END").As("origin_label"),
		goqu.I("l.name").As("location_name"),
	).
		From(goqu.T("non_serialized_items").As("s")).
		LeftJoin(goqu.T("origins").As("o"), goqu.On(goqu.Ex{"s.origin_id": goqu.I("o.id")})).
		LeftJoin(goqu.T("item_category").As("c"), goqu.On(goqu.Ex{"s.item_category_id": goqu.I("c.id")})).
		LeftJoin(goqu.T("locations").As("l"), goqu.On(goqu.Ex{"s.location_id": goqu.I("l.id")})).
		Where(goqu.I("s.id").In(stockIDs))

	type stockSnapshot struct {
		StockID        int     `db:"stock_id"`
		ItemCategoryID int     `db:"item_category_id"`
		Quantity       int     `db:"quantity"`
		CategoryName   *string `db:"category_name"`
		OriginLabel    *string `db:"origin_label"`
		LocationName   *string `db:"location_name"`
	}

	var snapshots []stockSnapshot
	if err := query.Executor().ScanStructs(&snapshots); err != nil {
		return fmt.Errorf("failed to fetch stock snapshots: %w", err)
	}

	if len(snapshots) != len(stocks) {
		return fmt.Errorf("some stocks not found: expected %d, found %d", len(stocks), len(snapshots))
	}

	rows := make([]interface{}, len(snapshots))
	for i, s := range snapshots {
		reqQty := qtyMap[s.StockID]
		if reqQty > s.Quantity {
			return fmt.Errorf("insufficient stock for id %d: requested %d, available %d", s.StockID, reqQty, s.Quantity)
		}
		rows[i] = goqu.Record{
			"release_id":       releaseID,
			"stock_id":         s.StockID,
			"item_category_id": s.ItemCategoryID,
			"category_name":    s.CategoryName,
			"quantity":         reqQty,
			"origin_label":     s.OriginLabel,
			"location_name":    s.LocationName,
		}
	}

	_, err := tx.Insert("release_stocks").Rows(rows...).Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to insert release stocks: %w", err)
	}
	return nil
}

// GetRelease returns a release by ID with origin label and creator name.
func (r *Repository) GetRelease(id int) (*Release, error) {
	var release Release
	found, err := r.repo.GoquDBWrapper.Select(
		goqu.I("r.id"),
		goqu.I("r.reference"),
		goqu.I("r.origin_id"),
		goqu.I("o.slug").As("origin_label"),
		goqu.I("r.notes"),
		goqu.I("r.status"),
		goqu.I("r.created_by"),
		goqu.I("u.username").As("created_by_name"),
		goqu.I("r.completed_at"),
		goqu.I("r.created_at"),
	).
		From(goqu.T("releases").As("r")).
		Join(goqu.T("origins").As("o"), goqu.On(goqu.Ex{"r.origin_id": goqu.I("o.id")})).
		LeftJoin(goqu.T("users").As("u"), goqu.On(goqu.Ex{"r.created_by": goqu.I("u.id")})).
		Where(goqu.Ex{"r.id": id}).
		Executor().ScanStruct(&release)
	if err != nil {
		return nil, fmt.Errorf("failed to get release: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &release, nil
}

// GetReleaseAssets returns all asset snapshots for a release.
func (r *Repository) GetReleaseAssets(releaseID int) ([]ReleaseAsset, error) {
	var assets []ReleaseAsset
	err := r.repo.GoquDBWrapper.Select("id", "release_id", "item_id", "pyr_code", "item_serial", "category_name", "origin_label", "location_name").
		From("release_assets").
		Where(goqu.Ex{"release_id": releaseID}).
		Order(goqu.I("id").Asc()).
		Executor().ScanStructs(&assets)
	if err != nil {
		return nil, fmt.Errorf("failed to get release assets: %w", err)
	}
	if assets == nil {
		assets = []ReleaseAsset{}
	}
	return assets, nil
}

// GetReleaseStocks returns all stock snapshots for a release.
func (r *Repository) GetReleaseStocks(releaseID int) ([]ReleaseStock, error) {
	var stocks []ReleaseStock
	err := r.repo.GoquDBWrapper.Select("id", "release_id", "stock_id", "item_category_id", "category_name", "quantity", "origin_label", "location_name").
		From("release_stocks").
		Where(goqu.Ex{"release_id": releaseID}).
		Order(goqu.I("id").Asc()).
		Executor().ScanStructs(&stocks)
	if err != nil {
		return nil, fmt.Errorf("failed to get release stocks: %w", err)
	}
	if stocks == nil {
		stocks = []ReleaseStock{}
	}
	return stocks, nil
}

// ListReleases returns releases with optional status and origin filters.
func (r *Repository) ListReleases(status *string, originID *int) ([]Release, error) {
	query := r.repo.GoquDBWrapper.Select(
		goqu.I("r.id"),
		goqu.I("r.reference"),
		goqu.I("r.origin_id"),
		goqu.I("o.slug").As("origin_label"),
		goqu.I("r.notes"),
		goqu.I("r.status"),
		goqu.I("r.created_by"),
		goqu.I("u.username").As("created_by_name"),
		goqu.I("r.completed_at"),
		goqu.I("r.created_at"),
	).
		From(goqu.T("releases").As("r")).
		Join(goqu.T("origins").As("o"), goqu.On(goqu.Ex{"r.origin_id": goqu.I("o.id")})).
		LeftJoin(goqu.T("users").As("u"), goqu.On(goqu.Ex{"r.created_by": goqu.I("u.id")})).
		Order(goqu.I("r.created_at").Desc())

	if status != nil {
		query = query.Where(goqu.Ex{"r.status": *status})
	}
	if originID != nil {
		query = query.Where(goqu.Ex{"r.origin_id": *originID})
	}

	var releases []Release
	if err := query.Executor().ScanStructs(&releases); err != nil {
		return nil, fmt.Errorf("failed to list releases: %w", err)
	}
	if releases == nil {
		releases = []Release{}
	}
	return releases, nil
}

// GenerateReference creates a reference like WYD-2026-001.
func (r *Repository) GenerateReference() (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("WYD-%d-", year)

	var maxRef *string
	_, err := r.repo.GoquDBWrapper.Select(goqu.MAX(goqu.I("reference"))).
		From("releases").
		Where(goqu.I("reference").Like(prefix + "%")).
		Executor().ScanVal(&maxRef)
	if err != nil {
		return "", fmt.Errorf("failed to generate reference: %w", err)
	}

	seq := 1
	if maxRef != nil {
		// Parse sequence number from "WYD-2026-003"
		_, _ = fmt.Sscanf(*maxRef, prefix+"%d", &seq)
		seq++
	}

	return fmt.Sprintf("WYD-%d-%03d", year, seq), nil
}

// ValidateAssetsForRelease checks that all asset IDs exist, are not in_transit,
// and are not in another draft release.
func (r *Repository) ValidateAssetsForRelease(itemIDs []int, excludeReleaseID *int) error {
	if len(itemIDs) == 0 {
		return nil
	}

	// Check all items exist and have valid status
	var validIDs []int
	err := r.repo.GoquDBWrapper.Select(goqu.I("id")).
		From("items").
		Where(
			goqu.I("id").In(itemIDs),
			goqu.I("status").In([]string{"available", "located"}),
		).
		Executor().ScanVals(&validIDs)
	if err != nil {
		return fmt.Errorf("failed to validate assets: %w", err)
	}
	if len(validIDs) != len(itemIDs) {
		return fmt.Errorf("some assets are not available for release (not found or in_transit): expected %d valid, found %d", len(itemIDs), len(validIDs))
	}

	// Check not in another draft release
	draftCheck := r.repo.GoquDBWrapper.Select(goqu.I("ra.item_id")).
		From(goqu.T("release_assets").As("ra")).
		Join(goqu.T("releases").As("rel"), goqu.On(goqu.Ex{"ra.release_id": goqu.I("rel.id")})).
		Where(
			goqu.I("ra.item_id").In(itemIDs),
			goqu.Ex{"rel.status": "draft"},
		)

	if excludeReleaseID != nil {
		draftCheck = draftCheck.Where(goqu.I("rel.id").Neq(*excludeReleaseID))
	}

	var conflicting []int
	if err := draftCheck.Executor().ScanVals(&conflicting); err != nil {
		return fmt.Errorf("failed to check draft conflicts: %w", err)
	}
	if len(conflicting) > 0 {
		return fmt.Errorf("assets already in another draft release: %v", conflicting)
	}

	return nil
}

// ValidateStocksForRelease checks that stock records exist and have sufficient quantity.
func (r *Repository) ValidateStocksForRelease(stocks []StockReleaseReq) error {
	if len(stocks) == 0 {
		return nil
	}

	stockIDs := make([]int, len(stocks))
	qtyMap := make(map[int]int)
	for i, s := range stocks {
		stockIDs[i] = s.StockID
		qtyMap[s.StockID] = s.Quantity
	}

	type stockRow struct {
		ID       int `db:"id"`
		Quantity int `db:"quantity"`
	}
	var rows []stockRow
	err := r.repo.GoquDBWrapper.Select("id", "quantity").
		From("non_serialized_items").
		Where(goqu.I("id").In(stockIDs)).
		Executor().ScanStructs(&rows)
	if err != nil {
		return fmt.Errorf("failed to validate stocks: %w", err)
	}
	if len(rows) != len(stocks) {
		return fmt.Errorf("some stocks not found: expected %d, found %d", len(stocks), len(rows))
	}

	for _, row := range rows {
		if qtyMap[row.ID] > row.Quantity {
			return fmt.Errorf("insufficient stock for id %d: requested %d, available %d", row.ID, qtyMap[row.ID], row.Quantity)
		}
	}

	return nil
}

// ClearReleaseItems removes all assets and stocks from a release (for update).
func (r *Repository) ClearReleaseItems(tx *goqu.TxDatabase, releaseID int) error {
	if _, err := tx.Delete("release_assets").Where(goqu.Ex{"release_id": releaseID}).Executor().Exec(); err != nil {
		return fmt.Errorf("failed to clear release assets: %w", err)
	}
	if _, err := tx.Delete("release_stocks").Where(goqu.Ex{"release_id": releaseID}).Executor().Exec(); err != nil {
		return fmt.Errorf("failed to clear release stocks: %w", err)
	}
	return nil
}

// ConfirmRelease executes the release: deletes assets, decreases stock, updates status.
func (r *Repository) ConfirmRelease(tx *goqu.TxDatabase, releaseID int) error {
	// 1. Refresh asset snapshots with latest data
	_, err := tx.Exec(`
		UPDATE release_assets ra SET
			pyr_code = i.pyr_code,
			item_serial = i.item_serial,
			category_name = c.label,
			origin_label = CASE WHEN i.origin_suffix IS NOT NULL THEN o.slug || '-' || i.origin_suffix ELSE o.slug END,
			location_name = l.name
		FROM items i
		LEFT JOIN origins o ON i.origin_id = o.id
		LEFT JOIN item_category c ON i.item_category_id = c.id
		LEFT JOIN locations l ON i.location_id = l.id
		WHERE ra.release_id = $1 AND ra.item_id = i.id
	`, releaseID)
	if err != nil {
		return fmt.Errorf("failed to refresh asset snapshots: %w", err)
	}

	// 2. Refresh stock snapshots
	_, err = tx.Exec(`
		UPDATE release_stocks rs SET
			category_name = c.label,
			origin_label = CASE WHEN s.origin_suffix IS NOT NULL THEN o.slug || '-' || s.origin_suffix ELSE o.slug END,
			location_name = l.name
		FROM non_serialized_items s
		LEFT JOIN origins o ON s.origin_id = o.id
		LEFT JOIN item_category c ON s.item_category_id = c.id
		LEFT JOIN locations l ON s.location_id = l.id
		WHERE rs.release_id = $1 AND rs.stock_id = s.id
	`, releaseID)
	if err != nil {
		return fmt.Errorf("failed to refresh stock snapshots: %w", err)
	}

	// 3. Delete serialized_transfers referencing these assets (safety)
	_, err = tx.Exec(`
		DELETE FROM serialized_transfers
		WHERE item_id IN (SELECT item_id FROM release_assets WHERE release_id = $1)
	`, releaseID)
	if err != nil {
		return fmt.Errorf("failed to clean serialized_transfers: %w", err)
	}

	// 4. Delete assets from items table
	_, err = tx.Exec(`
		DELETE FROM items
		WHERE id IN (SELECT item_id FROM release_assets WHERE release_id = $1)
	`, releaseID)
	if err != nil {
		return fmt.Errorf("failed to delete released assets: %w", err)
	}

	// 5. Decrease stock quantities
	_, err = tx.Exec(`
		UPDATE non_serialized_items s SET
			quantity = s.quantity - rs.quantity
		FROM release_stocks rs
		WHERE rs.release_id = $1 AND rs.stock_id = s.id
	`, releaseID)
	if err != nil {
		return fmt.Errorf("failed to decrease stock quantities: %w", err)
	}

	// 6. Delete zero-quantity stock records (except main warehouse location_id=1)
	_, err = tx.Exec(`
		DELETE FROM non_serialized_items
		WHERE quantity <= 0 AND location_id != 1
	`)
	if err != nil {
		return fmt.Errorf("failed to clean zero-quantity stocks: %w", err)
	}

	// 7. Update release status
	now := time.Now()
	_, err = tx.Update("releases").Set(goqu.Record{
		"status":       "completed",
		"completed_at": now,
	}).Where(goqu.Ex{"id": releaseID}).Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to update release status: %w", err)
	}

	return nil
}

// DeleteRelease removes a draft release and all its items (CASCADE).
func (r *Repository) DeleteRelease(releaseID int) error {
	result, err := r.repo.GoquDBWrapper.Delete("releases").
		Where(goqu.Ex{"id": releaseID, "status": "draft"}).
		Executor().Exec()
	if err != nil {
		return fmt.Errorf("failed to delete release: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("release not found or not in draft status")
	}
	return nil
}
