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
	LinkQuestToTransfer(ctx context.Context, questID string, transferID int) error
	GetQuestByTransferID(ctx context.Context, transferID int) (*Quest, error)
	UnlinkQuestFromTransfer(ctx context.Context, questID string) error
	FindStockItemsByCategory(fromLocationID int, categoryID int) ([]StockMatch, error)
	ResolveLocationByPavilionAndName(pavilion, name string) (*int, error)
	CreateSyncLog(ctx context.Context, log *SyncLog) error
	GetLatestSyncLog(ctx context.Context) (*SyncLog, error)
	GetCategoryMapping(ctx context.Context, itemName string) (*int, error)
	CreateCategoryMapping(ctx context.Context, mapping *CategoryMapping) error
	IncrementMappingUsage(ctx context.Context, itemName string) error
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
	ID           int       `db:"id" json:"id"`
	FormItemName string    `db:"form_item_name" json:"form_item_name"`
	CategoryID   int       `db:"category_id" json:"category_id"`
	CreatedBy    *int      `db:"created_by" json:"created_by,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	LastUsedAt   *time.Time `db:"last_used_at" json:"last_used_at,omitempty"`
	UseCount     int       `db:"use_count" json:"use_count"`
}

// QuestDB represents quest as stored in database
type QuestDB struct {
	ID                   int        `db:"id"`
	QuestKey             string     `db:"quest_key"`
	QuestID              string     `db:"quest_id"`
	DestinationPavilion  string     `db:"destination_pavilion"`
	DestinationLocation  string     `db:"destination_location"`
	Recipient            string     `db:"recipient"`
	DeliveryDate         time.Time  `db:"delivery_date"`
	PickupTime           *string    `db:"pickup_time"`
	BudgetOwner          *string    `db:"budget_owner"`
	Status               string     `db:"status"`
	TransferID           *int       `db:"transfer_id"`
	LastSyncedAt         time.Time  `db:"last_synced_at"`
	CreatedAt            time.Time  `db:"created_at"`
	CompletedAt          *time.Time `db:"completed_at"`
}

// ItemDB represents item as stored in database
type ItemDB struct {
	ID                      int      `db:"id"`
	QuestID                 int      `db:"quest_id"`
	ItemName                string   `db:"item_name"`
	Quantity                int      `db:"quantity"`
	CategoryID              *int     `db:"category_id"`
	CategoryMatchType       string   `db:"category_match_type"`
	CategoryMatchConfidence *float64 `db:"category_match_confidence"`
	BudgetOwner             *string  `db:"budget_owner"`
	Notes                   *string  `db:"notes"`
	SourceRowNumber         *int     `db:"source_row_number"`
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

// GetQuestByID retrieves quest with its items by quest_id (quest-abc123)
func (r *Repository) GetQuestByID(ctx context.Context, questID string) (*Quest, error) {
	var questDB QuestDB

	query := r.repo.GoquDBWrapper.
		Select("*").
		From("equipment_request_quests").
		Where(goqu.Ex{"quest_id": questID})

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

	return r.recordToQuest(&questDB, items), nil
}

// GetQuestByKey retrieves quest by quest_key hash
func (r *Repository) GetQuestByKey(ctx context.Context, questKey string) (*Quest, error) {
	var questDB QuestDB

	query := r.repo.GoquDBWrapper.
		Select("*").
		From("equipment_request_quests").
		Where(goqu.Ex{"quest_key": questKey})

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
	query := r.repo.GoquDBWrapper.
		Select("*").
		From("equipment_request_quests").
		Order(goqu.I("delivery_date").Asc(), goqu.I("created_at").Desc())

	// Apply filters
	if filter.Status != "" {
		query = query.Where(goqu.Ex{"status": filter.Status})
	}
	if filter.DeliveryAfter != nil {
		query = query.Where(goqu.I("delivery_date").Gte(*filter.DeliveryAfter))
	}
	if filter.DeliveryBefore != nil {
		query = query.Where(goqu.I("delivery_date").Lte(*filter.DeliveryBefore))
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

	_, err := r.repo.GoquDBWrapper.
		Update("equipment_request_quests").
		Set(updates).
		Where(goqu.Ex{"quest_id": questID}).
		Executor().Exec()

	if err != nil {
		return fmt.Errorf("failed to update quest status: %w", err)
	}

	return nil
}

// LinkQuestToTransfer links quest to a created transfer
func (r *Repository) LinkQuestToTransfer(ctx context.Context, questID string, transferID int) error {
	_, err := r.repo.GoquDBWrapper.
		Update("equipment_request_quests").
		Set(goqu.Record{
			"transfer_id": transferID,
			"status":      "in_progress",
		}).
		Where(goqu.Ex{"quest_id": questID}).
		Executor().Exec()

	if err != nil {
		return fmt.Errorf("failed to link quest to transfer: %w", err)
	}

	return nil
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
		"quest_key":             quest.QuestKey,
		"quest_id":              quest.ID,
		"destination_pavilion":  quest.Destination.Pavilion,
		"destination_location":  quest.Destination.Location,
		"recipient":             quest.Recipient,
		"delivery_date":         quest.DeliveryDate,
		"status":                quest.Status,
	}

	if quest.PickupTime != "" {
		record["pickup_time"] = quest.PickupTime
	}
	if quest.BudgetOwner != "" {
		record["budget_owner"] = quest.BudgetOwner
	}

	return record
}

// Helper: Convert QuestItem to database record
func (r *Repository) itemToRecord(questDBID int, item *QuestItem, sourceRow int) goqu.Record {
	record := goqu.Record{
		"quest_id":             questDBID,
		"item_name":            item.Name,
		"quantity":             item.Quantity,
		"category_match_type":  item.CategoryMatch,
		"source_row_number":    sourceRow,
	}

	if item.CategoryID != nil {
		record["category_id"] = *item.CategoryID
	}
	if item.CategoryMatchConfidence != 0.0 {
		record["category_match_confidence"] = item.CategoryMatchConfidence
	}
	if item.Notes != "" {
		record["notes"] = item.Notes
	}
	if item.BudgetOwner != "" {
		record["budget_owner"] = item.BudgetOwner
	}

	return record
}

// GetQuestByTransferID retrieves quest linked to a specific transfer
func (r *Repository) GetQuestByTransferID(ctx context.Context, transferID int) (*Quest, error) {
	var questDB QuestDB

	query := r.repo.GoquDBWrapper.
		Select("*").
		From("equipment_request_quests").
		Where(goqu.Ex{"transfer_id": transferID})

	found, err := query.Executor().ScanStruct(&questDB)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quest by transfer ID: %w", err)
	}
	if !found {
		return nil, nil // No quest linked to this transfer
	}

	items, err := r.getItemsByQuestDBID(ctx, questDB.ID)
	if err != nil {
		return nil, err
	}

	return r.recordToQuest(&questDB, items), nil
}

// UnlinkQuestFromTransfer removes transfer link and resets quest to pending
func (r *Repository) UnlinkQuestFromTransfer(ctx context.Context, questID string) error {
	_, err := r.repo.GoquDBWrapper.
		Update("equipment_request_quests").
		Set(goqu.Record{
			"transfer_id": nil,
			"status":      "pending",
		}).
		Where(goqu.Ex{"quest_id": questID}).
		Executor().Exec()

	if err != nil {
		return fmt.Errorf("failed to unlink quest from transfer: %w", err)
	}

	return nil
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

// Helper: Convert DB records to Quest
func (r *Repository) recordToQuest(questDB *QuestDB, itemsDB []ItemDB) *Quest {
	quest := &Quest{
		ID:       questDB.QuestID,
		QuestKey: questDB.QuestKey,
		Destination: Destination{
			Pavilion: questDB.DestinationPavilion,
			Location: questDB.DestinationLocation,
		},
		Recipient:    questDB.Recipient,
		DeliveryDate: questDB.DeliveryDate.Format("2006-01-02"),
		Status:       questDB.Status,
		TransferID:   questDB.TransferID,
		LastSynced:   questDB.LastSyncedAt,
		Items:        make([]QuestItem, len(itemsDB)),
		SourceRows:   make([]int, len(itemsDB)),
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
			Name:         itemDB.ItemName,
			Quantity:     itemDB.Quantity,
			CategoryID:   itemDB.CategoryID,
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
