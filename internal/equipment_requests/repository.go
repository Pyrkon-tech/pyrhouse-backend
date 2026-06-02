package equipment_requests

import (
	"context"
	"fmt"
	"time"

	"warehouse/internal/repository"

	"github.com/doug-martin/goqu/v9"
)

// QuestRepositoryInterface defines the contract for quest repository operations
type QuestRepositoryInterface interface {
	CreateQuest(ctx context.Context, quest *Quest) error
	UpdateQuest(ctx context.Context, questID string, quest *Quest) error
	GetQuestByID(ctx context.Context, questID string) (*Quest, error)
	GetQuestByKey(ctx context.Context, questKey string) (*Quest, error)
	ListQuests(ctx context.Context, filter QuestFilter) ([]Quest, error)
	UpdateQuestStatus(ctx context.Context, questID string, status string) error
	AddTransferToQuest(ctx context.Context, questID string, transferID int) error
	RemoveTransferFromQuest(ctx context.Context, transferID int) error
	GetQuestByTransferID(ctx context.Context, transferID int) (*Quest, error)
	GetActiveTransfersForQuest(ctx context.Context, questID string) ([]int, error)
	FindStockItemsByCategory(fromLocationID int, categoryID int) ([]StockMatch, error)
	ResolveLocationByPavilionAndName(pavilion, name string) (*int, error)
	ResolveLocationByNameOnly(name string) (*int, error)
	CreateSyncLog(ctx context.Context, log *SyncLog) error
	GetLatestSyncLog(ctx context.Context) (*SyncLog, error)
	GetCategoryMapping(ctx context.Context, itemName string) (*int, error)
	CreateCategoryMapping(ctx context.Context, mapping *CategoryMapping) error
	IncrementMappingUsage(ctx context.Context, itemName string) error
	ListCategoryMappings(ctx context.Context) ([]CategoryMapping, error)
	DeleteCategoryMapping(ctx context.Context, id int) error

	// Location mapping
	GetLocationMapping(ctx context.Context, pavilion, locationName string) (*int, error)
	CreateLocationMapping(ctx context.Context, mapping *LocationMapping) error
	ListLocationMappings(ctx context.Context) ([]LocationMapping, error)
	DeleteLocationMapping(ctx context.Context, id int) error
	IncrementLocationMappingUsage(ctx context.Context, pavilion, locationName string) error

	// Location resolution tracking
	UpdateQuestLocationResolution(ctx context.Context, questID string, locationID *int, resolved bool) error
	ListUnresolvedLocationQuests(ctx context.Context) ([]Quest, error)
}

// StockMatch represents a non-serialized stock item found at a location for a given category
type StockMatch struct {
	StockID      int    `db:"id"`
	CategoryID   int    `db:"item_category_id"`
	CategoryName string `db:"category_name"`
	Quantity     int    `db:"quantity"`
	LocationID   int    `db:"location_id"`
}

type Repository struct {
	repo *repository.Repository
}

func NewRepository(repo *repository.Repository) *Repository {
	return &Repository{repo: repo}
}

// QuestFilter for listing quests with pagination and filtering
type QuestFilter struct {
	Status         string
	LocationID     *int
	Limit          int
	Offset         int
	DeliveryAfter  *time.Time
	DeliveryBefore *time.Time
}

// SyncLog represents a sync operation record
type SyncLog struct {
	ID              int       `db:"id" json:"id"`
	SyncedAt        time.Time `db:"synced_at" json:"synced_at"`
	RowsProcessed   int       `db:"rows_processed" json:"rows_processed"`
	QuestsCreated   int       `db:"quests_created" json:"quests_created"`
	QuestsUpdated   int       `db:"quests_updated" json:"quests_updated"`
	QuestsUnchanged int       `db:"quests_unchanged" json:"quests_unchanged"`
	ItemsAdded      int       `db:"items_added" json:"items_added"`
	ItemsRemoved    int       `db:"items_removed" json:"items_removed"`
	Errors          string    `db:"errors" json:"errors,omitempty"`
	Success         bool      `db:"success" json:"success"`
	DurationMs      int       `db:"duration_ms" json:"duration_ms"`
	SheetID         string    `db:"sheet_id" json:"sheet_id"`
}

// CategoryMapping represents manual item-to-category mapping
type CategoryMapping struct {
	ID           int        `db:"id" json:"id"`
	FormItemName string     `db:"form_item_name" json:"form_item_name"`
	CategoryID   int        `db:"category_id" json:"category_id"`
	CreatedBy    *int       `db:"created_by" json:"created_by,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	LastUsedAt   *time.Time `db:"last_used_at" json:"last_used_at,omitempty"`
	UseCount     int        `db:"use_count" json:"use_count"`
}

// QuestDB represents quest as stored in database
type QuestDB struct {
	ID                  int        `db:"id"`
	QuestKey            string     `db:"quest_key"`
	QuestID             string     `db:"quest_id"`
	DestinationPavilion string     `db:"destination_pavilion"`
	DestinationLocation string     `db:"destination_location"`
	Recipient           string     `db:"recipient"`
	DeliveryDate        time.Time  `db:"delivery_date"`
	PickupTime          *string    `db:"pickup_time"`
	BudgetOwner         *string    `db:"budget_owner"`
	Status              string     `db:"status"`
	LocationID          *int       `db:"location_id"`
	LocationName        *string    `db:"location_name"`
	LocationResolved    bool       `db:"location_resolved"`
	LastSyncedAt        time.Time  `db:"last_synced_at"`
	CreatedAt           time.Time  `db:"created_at"`
	CompletedAt         *time.Time `db:"completed_at"`
}

// ItemDB represents item as stored in database
type ItemDB struct {
	ID                      int       `db:"id"`
	QuestID                 int       `db:"quest_id"`
	ItemName                string    `db:"item_name"`
	Quantity                int       `db:"quantity"`
	CategoryID              *int      `db:"category_id"`
	CategoryMatchType       string    `db:"category_match_type"`
	CategoryMatchConfidence *float64  `db:"category_match_confidence"`
	BudgetOwner             *string   `db:"budget_owner"`
	Notes                   *string   `db:"notes"`
	SourceRowNumber         *int      `db:"source_row_number"`
	CreatedAt               time.Time `db:"created_at"`
}

// CreateQuest creates a new quest with its items in a transaction
func (r *Repository) CreateQuest(ctx context.Context, quest *Quest) error {
	return repository.WithTransaction(r.repo.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		// 1. Insert quest
		questRecord := r.questToRecord(quest)

		var questDBID int
		insertQuery := tx.Insert("equipment_request_quests").
			Rows(questRecord).
			Returning("id")

		if _, err := insertQuery.Executor().ScanVal(&questDBID); err != nil {
			return fmt.Errorf("failed to create quest: %w", err)
		}

		// 2. Insert items
		if len(quest.Items) > 0 {
			itemRecords := make([]interface{}, 0, len(quest.Items))
			for i := range quest.Items {
				itemRecord := r.itemToRecord(questDBID, &quest.Items[i], quest.SourceRows[i])
				itemRecords = append(itemRecords, itemRecord)
			}

			if _, err := tx.Insert("equipment_request_items").Rows(itemRecords...).Executor().Exec(); err != nil {
				return fmt.Errorf("failed to create quest items: %w", err)
			}
		}

		return nil
	})
}

// UpdateQuest updates existing quest and replaces its items
func (r *Repository) UpdateQuest(ctx context.Context, questID string, quest *Quest) error {
	return repository.WithTransaction(r.repo.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		// Get DB ID first
		var questDBID int
		found, err := tx.Select("id").
			From("equipment_request_quests").
			Where(goqu.Ex{"quest_id": questID}).
			Executor().ScanVal(&questDBID)

		if err != nil || !found {
			return fmt.Errorf("quest not found: %s", questID)
		}

		// 1. Update quest
		questRecord := r.questToRecord(quest)
		questRecord["last_synced_at"] = time.Now()

		if _, err := tx.Update("equipment_request_quests").
			Set(questRecord).
			Where(goqu.Ex{"quest_id": questID}).
			Executor().Exec(); err != nil {
			return fmt.Errorf("failed to update quest: %w", err)
		}

		// 2. Delete old items
		if _, err := tx.Delete("equipment_request_items").
			Where(goqu.Ex{"quest_id": questDBID}).
			Executor().Exec(); err != nil {
			return fmt.Errorf("failed to delete old items: %w", err)
		}

		// 3. Insert new items
		if len(quest.Items) > 0 {
			itemRecords := make([]interface{}, 0, len(quest.Items))
			for i := range quest.Items {
				itemRecord := r.itemToRecord(questDBID, &quest.Items[i], quest.SourceRows[i])
				itemRecords = append(itemRecords, itemRecord)
			}

			if _, err := tx.Insert("equipment_request_items").Rows(itemRecords...).Executor().Exec(); err != nil {
				return fmt.Errorf("failed to create new items: %w", err)
			}
		}

		return nil
	})
}

// questBaseQuery returns the base SELECT with LEFT JOIN on locations for location_name.
func (r *Repository) questBaseQuery() *goqu.SelectDataset {
	return r.repo.GoquDBWrapper.
		Select(
			goqu.I("q.*"),
			goqu.I("l.name").As("location_name"),
		).
		From(goqu.T("equipment_request_quests").As("q")).
		LeftJoin(
			goqu.T("locations").As("l"),
			goqu.On(goqu.Ex{"q.location_id": goqu.I("l.id")}),
		)
}

// GetQuestByID retrieves quest with its items by quest_id (quest-abc123)
func (r *Repository) GetQuestByID(ctx context.Context, questID string) (*Quest, error) {
	var questDB QuestDB

	query := r.questBaseQuery().
		Where(goqu.Ex{"q.quest_id": questID})

	found, err := query.Executor().ScanStruct(&questDB)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quest: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("quest not found")
	}

	// Fetch items
	items, err := r.getItemsByQuestDBID(ctx, questDB.ID)
	if err != nil {
		return nil, err
	}

	quest := r.recordToQuest(&questDB, items)

	// Load all linked transfers
	transfers, err := r.getQuestTransfers(ctx, questID)
	if err != nil {
		return nil, err
	}
	quest.Transfers = transfers

	// Aggregate volunteers across all linked transfers
	if len(transfers) > 0 {
		ids := make([]int, len(transfers))
		for i, t := range transfers {
			ids[i] = t.TransferID
		}
		volunteers, err := r.getQuestVolunteers(ctx, ids)
		if err != nil {
			return nil, err
		}
		quest.AssignedVolunteers = volunteers
	}

	return quest, nil
}

// getQuestVolunteers returns unique users linked to any of the given transfers via transfer_users.
func (r *Repository) getQuestVolunteers(_ context.Context, transferIDs []int) ([]QuestVolunteer, error) {
	if len(transferIDs) == 0 {
		return []QuestVolunteer{}, nil
	}

	query := r.repo.GoquDBWrapper.
		Select(
			goqu.I("u.id"),
			goqu.I("u.username"),
			goqu.I("u.fullname"),
		).
		From(goqu.T("transfer_users").As("tu")).
		Join(goqu.T("users").As("u"), goqu.On(goqu.Ex{"tu.user_id": goqu.I("u.id")})).
		Where(goqu.I("tu.transfer_id").In(transferIDs)).
		Order(goqu.I("u.id").Asc())

	rows, err := query.Executor().Query()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quest volunteers: %w", err)
	}
	defer rows.Close()

	seen := make(map[int]struct{})
	var volunteers []QuestVolunteer
	for rows.Next() {
		var v QuestVolunteer
		if err := rows.Scan(&v.ID, &v.Username, &v.Fullname); err != nil {
			return nil, fmt.Errorf("failed to scan volunteer: %w", err)
		}
		if _, dup := seen[v.ID]; !dup {
			seen[v.ID] = struct{}{}
			volunteers = append(volunteers, v)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row error: %w", err)
	}
	if volunteers == nil {
		volunteers = []QuestVolunteer{}
	}
	return volunteers, nil
}

// getQuestTransfers returns all transfers linked to a quest, with their current status.
func (r *Repository) getQuestTransfers(_ context.Context, questID string) ([]QuestTransfer, error) {
	var rows []QuestTransfer

	query := r.repo.GoquDBWrapper.
		Select(
			goqu.I("qt.transfer_id"),
			goqu.I("t.status").As("transfer_status"),
			goqu.I("qt.created_at"),
		).
		From(goqu.T("quest_transfers").As("qt")).
		Join(
			goqu.T("transfers").As("t"),
			goqu.On(goqu.Ex{"qt.transfer_id": goqu.I("t.id")}),
		).
		Where(goqu.Ex{"qt.quest_id": questID}).
		Order(goqu.I("qt.created_at").Asc())

	if err := query.Executor().ScanStructs(&rows); err != nil {
		return nil, fmt.Errorf("failed to fetch quest transfers: %w", err)
	}
	if rows == nil {
		rows = []QuestTransfer{}
	}
	return rows, nil
}

// GetQuestByKey retrieves quest by quest_key hash
func (r *Repository) GetQuestByKey(ctx context.Context, questKey string) (*Quest, error) {
	var questDB QuestDB

	query := r.questBaseQuery().
		Where(goqu.Ex{"q.quest_key": questKey})

	found, err := query.Executor().ScanStruct(&questDB)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quest by key: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("quest not found")
	}

	// Fetch items
	items, err := r.getItemsByQuestDBID(ctx, questDB.ID)
	if err != nil {
		return nil, err
	}

	return r.recordToQuest(&questDB, items), nil
}

// ListQuests retrieves quests with filtering and pagination
func (r *Repository) ListQuests(ctx context.Context, filter QuestFilter) ([]Quest, error) {
	query := r.questBaseQuery().
		Order(
			goqu.L("CASE q.status WHEN 'pending' THEN 0 WHEN 'in_progress' THEN 1 WHEN 'completed' THEN 2 ELSE 3 END").Asc(),
			goqu.L("CASE WHEN q.status IN ('pending', 'in_progress') THEN q.delivery_date END").Asc(),
			goqu.L("CASE WHEN q.status = 'completed' THEN q.completed_at END").Desc().NullsLast(),
		)

	// Apply filters
	if filter.Status != "" {
		query = query.Where(goqu.Ex{"q.status": filter.Status})
	}
	if filter.LocationID != nil {
		query = query.Where(goqu.Ex{"q.location_id": *filter.LocationID})
	}
	if filter.DeliveryAfter != nil {
		query = query.Where(goqu.I("q.delivery_date").Gte(*filter.DeliveryAfter))
	}
	if filter.DeliveryBefore != nil {
		query = query.Where(goqu.I("q.delivery_date").Lte(*filter.DeliveryBefore))
	}

	// Pagination
	if filter.Limit > 0 {
		query = query.Limit(uint(filter.Limit))
	}
	if filter.Offset > 0 {
		query = query.Offset(uint(filter.Offset))
	}

	var questsDB []QuestDB
	if err := query.Executor().ScanStructs(&questsDB); err != nil {
		return nil, fmt.Errorf("failed to list quests: %w", err)
	}

	// Fetch items for each quest
	quests := make([]Quest, len(questsDB))
	for i, questDB := range questsDB {
		items, err := r.getItemsByQuestDBID(ctx, questDB.ID)
		if err != nil {
			return nil, err
		}
		quests[i] = *r.recordToQuest(&questDB, items)
	}

	return quests, nil
}

// UpdateQuestStatus updates quest status and optionally sets completed_at
func (r *Repository) UpdateQuestStatus(ctx context.Context, questID string, status string) error {
	updates := goqu.Record{"status": status}

	if status == "completed" {
		now := time.Now()
		updates["completed_at"] = now
	}

	result, err := r.repo.GoquDBWrapper.
		Update("equipment_request_quests").
		Set(updates).
		Where(goqu.Ex{"quest_id": questID}).
		Executor().Exec()

	if err != nil {
		return fmt.Errorf("failed to update quest status: %w", err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("quest not found: %s", questID)
	}

	return nil
}

// AddTransferToQuest inserts a row into quest_transfers and sets quest status to in_progress.
func (r *Repository) AddTransferToQuest(ctx context.Context, questID string, transferID int) error {
	return repository.WithTransaction(r.repo.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		if _, err := tx.Insert("quest_transfers").
			Rows(goqu.Record{
				"quest_id":    questID,
				"transfer_id": transferID,
			}).
			Executor().Exec(); err != nil {
			return fmt.Errorf("failed to insert quest_transfer: %w", err)
		}

		result, err := tx.Update("equipment_request_quests").
			Set(goqu.Record{"status": "in_progress"}).
			Where(goqu.Ex{"quest_id": questID, "status": "pending"}).
			Executor().Exec()
		if err != nil {
			return fmt.Errorf("failed to set quest in_progress: %w", err)
		}
		// RowsAffected == 0 is fine — quest was already in_progress
		_ = result
		return nil
	})
}

// CreateSyncLog creates a sync history record
func (r *Repository) CreateSyncLog(ctx context.Context, log *SyncLog) error {
	record := goqu.Record{
		"rows_processed":   log.RowsProcessed,
		"quests_created":   log.QuestsCreated,
		"quests_updated":   log.QuestsUpdated,
		"quests_unchanged": log.QuestsUnchanged,
		"items_added":      log.ItemsAdded,
		"items_removed":    log.ItemsRemoved,
		"errors":           log.Errors,
		"success":          log.Success,
		"duration_ms":      log.DurationMs,
		"sheet_id":         log.SheetID,
	}

	_, err := r.repo.GoquDBWrapper.
		Insert("equipment_request_sync_log").
		Rows(record).
		Executor().Exec()

	if err != nil {
		return fmt.Errorf("failed to create sync log: %w", err)
	}

	return nil
}

// GetLatestSyncLog retrieves the most recent sync log
func (r *Repository) GetLatestSyncLog(ctx context.Context) (*SyncLog, error) {
	var log SyncLog

	query := r.repo.GoquDBWrapper.
		Select("*").
		From("equipment_request_sync_log").
		Order(goqu.I("synced_at").Desc()).
		Limit(1)

	found, err := query.Executor().ScanStruct(&log)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest sync log: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("no sync log found")
	}

	return &log, nil
}

// GetCategoryMapping retrieves manual category mapping for an item name
func (r *Repository) GetCategoryMapping(ctx context.Context, itemName string) (*int, error) {
	var mapping CategoryMapping

	query := r.repo.GoquDBWrapper.
		Select("*").
		From("equipment_request_category_mapping").
		Where(goqu.Ex{"form_item_name": itemName})

	found, err := query.Executor().ScanStruct(&mapping)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch category mapping: %w", err)
	}
	if !found {
		return nil, nil // No mapping found is not an error
	}

	return &mapping.CategoryID, nil
}

// CreateCategoryMapping creates a manual category mapping
func (r *Repository) CreateCategoryMapping(ctx context.Context, mapping *CategoryMapping) error {
	record := goqu.Record{
		"form_item_name": mapping.FormItemName,
		"category_id":    mapping.CategoryID,
		"created_by":     mapping.CreatedBy,
	}

	_, err := r.repo.GoquDBWrapper.
		Insert("equipment_request_category_mapping").
		Rows(record).
		Executor().Exec()

	if err != nil {
		return fmt.Errorf("failed to create category mapping: %w", err)
	}

	return nil
}

// IncrementMappingUsage updates last_used_at and increments use_count
func (r *Repository) IncrementMappingUsage(ctx context.Context, itemName string) error {
	now := time.Now()

	_, err := r.repo.GoquDBWrapper.
		Update("equipment_request_category_mapping").
		Set(goqu.Record{
			"last_used_at": now,
			"use_count":    goqu.L("use_count + 1"),
		}).
		Where(goqu.Ex{"form_item_name": itemName}).
		Executor().Exec()

	if err != nil {
		return fmt.Errorf("failed to increment mapping usage: %w", err)
	}

	return nil
}

// ListCategoryMappings returns all manual category mappings ordered by use frequency
func (r *Repository) ListCategoryMappings(ctx context.Context) ([]CategoryMapping, error) {
	var mappings []CategoryMapping

	query := r.repo.GoquDBWrapper.
		Select("*").
		From("equipment_request_category_mapping").
		Order(
			goqu.I("use_count").Desc(),
			goqu.I("created_at").Desc(),
		)

	if err := query.Executor().ScanStructs(&mappings); err != nil {
		return nil, fmt.Errorf("failed to list category mappings: %w", err)
	}

	return mappings, nil
}

// DeleteCategoryMapping removes a manual category mapping by ID
func (r *Repository) DeleteCategoryMapping(ctx context.Context, id int) error {
	result, err := r.repo.GoquDBWrapper.
		Delete("equipment_request_category_mapping").
		Where(goqu.Ex{"id": id}).
		Executor().Exec()

	if err != nil {
		return fmt.Errorf("failed to delete category mapping: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("category mapping %d not found", id)
	}

	return nil
}

// Helper: Get items by quest DB ID
func (r *Repository) getItemsByQuestDBID(ctx context.Context, questDBID int) ([]ItemDB, error) {
	var items []ItemDB

	query := r.repo.GoquDBWrapper.
		Select("*").
		From("equipment_request_items").
		Where(goqu.Ex{"quest_id": questDBID}).
		Order(goqu.I("id").Asc())

	if err := query.Executor().ScanStructs(&items); err != nil {
		return nil, fmt.Errorf("failed to fetch quest items: %w", err)
	}

	return items, nil
}

// Helper: Convert Quest to database record
func (r *Repository) questToRecord(quest *Quest) goqu.Record {
	record := goqu.Record{
		"quest_key":            quest.QuestKey,
		"quest_id":             quest.ID,
		"destination_pavilion": quest.Destination.Pavilion,
		"destination_location": quest.Destination.Location,
		"recipient":            quest.Recipient,
		"status":               quest.Status,
		"location_resolved":    quest.LocationResolved,
	}
	if quest.DeliveryDate != "" {
		record["delivery_date"] = quest.DeliveryDate
	}

	if quest.PickupTime != "" {
		record["pickup_time"] = quest.PickupTime
	}
	if quest.BudgetOwner != "" {
		record["budget_owner"] = quest.BudgetOwner
	}
	if quest.LocationID != nil {
		record["location_id"] = *quest.LocationID
	} else {
		record["location_id"] = nil
	}

	return record
}

// Helper: Convert QuestItem to database record.
// All columns must be present for batch insert — goqu requires identical keys across rows.
func (r *Repository) itemToRecord(questDBID int, item *QuestItem, sourceRow int) goqu.Record {
	record := goqu.Record{
		"quest_id":            questDBID,
		"item_name":           item.Name,
		"quantity":            item.Quantity,
		"category_match_type": item.CategoryMatch,
		"source_row_number":   sourceRow,
	}

	var catID interface{}
	if item.CategoryID != nil {
		catID = *item.CategoryID
	}
	record["category_id"] = catID

	var conf interface{}
	if item.CategoryMatchConfidence != 0.0 {
		conf = item.CategoryMatchConfidence
	}
	record["category_match_confidence"] = conf

	var notes interface{}
	if item.Notes != "" {
		notes = item.Notes
	}
	record["notes"] = notes

	var budgetOwner interface{}
	if item.BudgetOwner != "" {
		budgetOwner = item.BudgetOwner
	}
	record["budget_owner"] = budgetOwner

	return record
}

// GetQuestByTransferID retrieves the quest linked to a specific transfer via quest_transfers.
func (r *Repository) GetQuestByTransferID(ctx context.Context, transferID int) (*Quest, error) {
	var questDB QuestDB

	query := r.questBaseQuery().
		Join(
			goqu.T("quest_transfers").As("qt"),
			goqu.On(goqu.Ex{"qt.quest_id": goqu.I("q.quest_id")}),
		).
		Where(goqu.Ex{"qt.transfer_id": transferID})

	found, err := query.Executor().ScanStruct(&questDB)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quest by transfer ID: %w", err)
	}
	if !found {
		return nil, nil
	}

	items, err := r.getItemsByQuestDBID(ctx, questDB.ID)
	if err != nil {
		return nil, err
	}

	return r.recordToQuest(&questDB, items), nil
}

// RemoveTransferFromQuest deletes the quest_transfers row for the given transfer.
// If no active transfers remain for the quest, it resets quest status to pending.
func (r *Repository) RemoveTransferFromQuest(ctx context.Context, transferID int) error {
	return repository.WithTransaction(r.repo.GoquDBWrapper, func(tx *goqu.TxDatabase) error {
		// Identify the quest before deletion
		var questID string
		found, err := tx.Select("quest_id").
			From("quest_transfers").
			Where(goqu.Ex{"transfer_id": transferID}).
			Executor().ScanVal(&questID)
		if err != nil {
			return fmt.Errorf("failed to find quest for transfer %d: %w", transferID, err)
		}
		if !found {
			return nil // already gone
		}

		if _, err := tx.Delete("quest_transfers").
			Where(goqu.Ex{"transfer_id": transferID}).
			Executor().Exec(); err != nil {
			return fmt.Errorf("failed to delete quest_transfer: %w", err)
		}

		// Count remaining active transfers for this quest
		var remaining int
		if _, err := tx.Select(goqu.COUNT("qt.transfer_id")).
			From(goqu.T("quest_transfers").As("qt")).
			Join(goqu.T("transfers").As("t"), goqu.On(goqu.Ex{"qt.transfer_id": goqu.I("t.id")})).
			Where(
				goqu.Ex{"qt.quest_id": questID},
				goqu.I("t.status").NotIn("completed", "cancelled"),
			).
			Executor().ScanVal(&remaining); err != nil {
			return fmt.Errorf("failed to count active transfers: %w", err)
		}

		if remaining == 0 {
			if _, err := tx.Update("equipment_request_quests").
				Set(goqu.Record{"status": "pending"}).
				Where(goqu.Ex{"quest_id": questID, "status": "in_progress"}).
				Executor().Exec(); err != nil {
				return fmt.Errorf("failed to reset quest to pending: %w", err)
			}
		}
		return nil
	})
}

// GetActiveTransfersForQuest returns IDs of non-completed, non-cancelled transfers for the quest.
func (r *Repository) GetActiveTransfersForQuest(ctx context.Context, questID string) ([]int, error) {
	var rows []struct {
		TransferID int `db:"transfer_id"`
	}

	query := r.repo.GoquDBWrapper.
		Select(goqu.I("qt.transfer_id")).
		From(goqu.T("quest_transfers").As("qt")).
		Join(
			goqu.T("transfers").As("t"),
			goqu.On(goqu.Ex{"qt.transfer_id": goqu.I("t.id")}),
		).
		Where(
			goqu.Ex{"qt.quest_id": questID},
			goqu.I("t.status").NotIn("completed", "cancelled"),
		)

	if err := query.Executor().ScanStructs(&rows); err != nil {
		return nil, fmt.Errorf("failed to get active transfers for quest %s: %w", questID, err)
	}

	ids := make([]int, len(rows))
	for i, row := range rows {
		ids[i] = row.TransferID
	}
	return ids, nil
}

// FindStockItemsByCategory finds non-serialized stock items at a location matching a category
func (r *Repository) FindStockItemsByCategory(fromLocationID int, categoryID int) ([]StockMatch, error) {
	var matches []StockMatch

	query := r.repo.GoquDBWrapper.
		Select(
			goqu.I("s.id"),
			goqu.I("s.item_category_id"),
			goqu.I("c.item_category").As("category_name"),
			goqu.I("s.quantity"),
			goqu.I("s.location_id"),
		).
		From(goqu.T("non_serialized_items").As("s")).
		LeftJoin(
			goqu.T("item_category").As("c"),
			goqu.On(goqu.Ex{"s.item_category_id": goqu.I("c.id")}),
		).
		Where(goqu.Ex{
			"s.location_id":      fromLocationID,
			"s.item_category_id": categoryID,
		}).
		Where(goqu.I("s.quantity").Gt(0))

	if err := query.Executor().ScanStructs(&matches); err != nil {
		return nil, fmt.Errorf("failed to find stock items by category: %w", err)
	}

	return matches, nil
}

// ResolveLocationByPavilionAndName finds a location ID by pavilion and name (case-insensitive)
func (r *Repository) ResolveLocationByPavilionAndName(pavilion, name string) (*int, error) {
	var locationID int

	query := r.repo.GoquDBWrapper.
		Select("id").
		From("locations").
		Where(
			goqu.I("pavilion").ILike(pavilion),
			goqu.I("name").ILike(name),
		).
		Order(goqu.I("id").Asc()).
		Limit(1)

	found, err := query.Executor().ScanVal(&locationID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve location: %w", err)
	}
	if !found {
		return nil, nil
	}

	return &locationID, nil
}

// ResolveLocationByNameOnly finds a location ID by name only; returns result only if exactly one match.
func (r *Repository) ResolveLocationByNameOnly(name string) (*int, error) {
	var rows []struct {
		ID int `db:"id"`
	}

	query := r.repo.GoquDBWrapper.
		Select("id").
		From("locations").
		Where(goqu.I("name").ILike(name)).
		Limit(2)

	if err := query.Executor().ScanStructs(&rows); err != nil {
		return nil, fmt.Errorf("failed to resolve location by name: %w", err)
	}
	if len(rows) != 1 {
		return nil, nil
	}
	return &rows[0].ID, nil
}

// GetLocationMapping retrieves manual location mapping for pavilion + location_name
func (r *Repository) GetLocationMapping(ctx context.Context, pavilion, locationName string) (*int, error) {
	var mapping LocationMapping

	query := r.repo.GoquDBWrapper.
		Select("*").
		From("equipment_request_location_mapping").
		Where(
			goqu.Ex{"pavilion": pavilion, "location_name": locationName},
		)

	found, err := query.Executor().ScanStruct(&mapping)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch location mapping: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &mapping.LocationID, nil
}

// CreateLocationMapping creates a manual location mapping and populates id, created_at on the struct
func (r *Repository) CreateLocationMapping(ctx context.Context, mapping *LocationMapping) error {
	record := goqu.Record{
		"pavilion":      mapping.Pavilion,
		"location_name": mapping.LocationName,
		"location_id":   mapping.LocationID,
	}

	query := r.repo.GoquDBWrapper.
		Insert("equipment_request_location_mapping").
		Rows(record).
		Returning("id", "created_at", "usage_count")

	_, err := query.Executor().ScanStruct(mapping)
	if err != nil {
		return fmt.Errorf("failed to create location mapping: %w", err)
	}
	return nil
}

// ListLocationMappings returns all manual location mappings ordered by usage
func (r *Repository) ListLocationMappings(ctx context.Context) ([]LocationMapping, error) {
	var mappings []LocationMapping

	query := r.repo.GoquDBWrapper.
		Select("*").
		From("equipment_request_location_mapping").
		Order(
			goqu.I("usage_count").Desc(),
			goqu.I("created_at").Desc(),
		)

	if err := query.Executor().ScanStructs(&mappings); err != nil {
		return nil, fmt.Errorf("failed to list location mappings: %w", err)
	}
	return mappings, nil
}

// DeleteLocationMapping removes a manual location mapping by ID
func (r *Repository) DeleteLocationMapping(ctx context.Context, id int) error {
	result, err := r.repo.GoquDBWrapper.
		Delete("equipment_request_location_mapping").
		Where(goqu.Ex{"id": id}).
		Executor().Exec()

	if err != nil {
		return fmt.Errorf("failed to delete location mapping: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("location mapping %d not found", id)
	}
	return nil
}

// IncrementLocationMappingUsage updates usage_count for a mapping
func (r *Repository) IncrementLocationMappingUsage(ctx context.Context, pavilion, locationName string) error {
	_, err := r.repo.GoquDBWrapper.
		Update("equipment_request_location_mapping").
		Set(goqu.Record{"usage_count": goqu.L("usage_count + 1")}).
		Where(goqu.Ex{"pavilion": pavilion, "location_name": locationName}).
		Executor().Exec()

	if err != nil {
		return fmt.Errorf("failed to increment location mapping usage: %w", err)
	}
	return nil
}

// UpdateQuestLocationResolution sets location_id and location_resolved for a quest
func (r *Repository) UpdateQuestLocationResolution(ctx context.Context, questID string, locationID *int, resolved bool) error {
	updates := goqu.Record{"location_resolved": resolved}
	if locationID != nil {
		updates["location_id"] = *locationID
	} else {
		updates["location_id"] = nil
	}

	_, err := r.repo.GoquDBWrapper.
		Update("equipment_request_quests").
		Set(updates).
		Where(goqu.Ex{"quest_id": questID}).
		Executor().Exec()

	if err != nil {
		return fmt.Errorf("failed to update quest location resolution: %w", err)
	}
	return nil
}

// ListUnresolvedLocationQuests returns quests with location_resolved = false
func (r *Repository) ListUnresolvedLocationQuests(ctx context.Context) ([]Quest, error) {
	query := r.questBaseQuery().
		Where(goqu.Ex{"q.location_resolved": false}).
		Order(goqu.I("q.delivery_date").Asc(), goqu.I("q.created_at").Asc())

	var questsDB []QuestDB
	if err := query.Executor().ScanStructs(&questsDB); err != nil {
		return nil, fmt.Errorf("failed to list unresolved location quests: %w", err)
	}

	quests := make([]Quest, len(questsDB))
	for i, questDB := range questsDB {
		items, err := r.getItemsByQuestDBID(ctx, questDB.ID)
		if err != nil {
			return nil, err
		}
		quests[i] = *r.recordToQuest(&questDB, items)
	}
	return quests, nil
}

// Helper: Convert DB records to Quest
func (r *Repository) recordToQuest(questDB *QuestDB, itemsDB []ItemDB) *Quest {
	quest := &Quest{
		ID:                 questDB.QuestID,
		QuestKey:           questDB.QuestKey,
		Destination:        Destination{Pavilion: questDB.DestinationPavilion, Location: questDB.DestinationLocation},
		Recipient:          questDB.Recipient,
		DeliveryDate:       questDB.DeliveryDate.Format("2006-01-02"),
		Status:             questDB.Status,
		LocationID:         questDB.LocationID,
		LocationName:       questDB.LocationName,
		LocationResolved:   questDB.LocationResolved,
		LastSynced:         questDB.LastSyncedAt,
		Items:              make([]QuestItem, len(itemsDB)),
		SourceRows:         make([]int, len(itemsDB)),
		Transfers:          []QuestTransfer{},
		AssignedVolunteers: []QuestVolunteer{},
	}

	if questDB.PickupTime != nil {
		quest.PickupTime = *questDB.PickupTime
	}
	if questDB.BudgetOwner != nil {
		quest.BudgetOwner = *questDB.BudgetOwner
	}

	// Convert items
	for i, itemDB := range itemsDB {
		quest.Items[i] = QuestItem{
			Name:          itemDB.ItemName,
			Quantity:      itemDB.Quantity,
			CategoryID:    itemDB.CategoryID,
			CategoryMatch: itemDB.CategoryMatchType,
		}

		if itemDB.CategoryMatchConfidence != nil {
			quest.Items[i].CategoryMatchConfidence = *itemDB.CategoryMatchConfidence
		}
		if itemDB.Notes != nil {
			quest.Items[i].Notes = *itemDB.Notes
		}
		if itemDB.BudgetOwner != nil {
			quest.Items[i].BudgetOwner = *itemDB.BudgetOwner
		}
		if itemDB.SourceRowNumber != nil {
			quest.SourceRows[i] = *itemDB.SourceRowNumber
		}
	}

	return quest
}
