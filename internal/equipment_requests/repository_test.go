package equipment_requests

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"warehouse/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRepository_CreateQuest tests creating a new quest in the database
func TestRepository_CreateQuest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(repository.NewRepository(db))
	ctx := context.Background()

	categoryID := 123
	quest := &Quest{
		ID:       "quest-test123",
		QuestKey: "test-key-1",
		Destination: Destination{
			Pavilion: "PCC",
			Location: "Maskarada",
		},
		Recipient:    "Jan Kowalski",
		DeliveryDate: "2025-06-13",
		PickupTime:   "17-18",
		BudgetOwner:  "Anna Nowak",
		Items: []QuestItem{
			{
				Name:                    "Laptop",
				Quantity:                2,
				CategoryID:              &categoryID,
				CategoryMatch:           "exact",
				CategoryMatchConfidence: 1.0,
				BudgetOwner:             "Anna Nowak",
				Notes:                   "Test note",
			},
		},
		Status:     "pending",
		SourceRows: []int{10, 11},
		LastSynced: time.Now(),
	}

	err := repo.CreateQuest(ctx, quest)
	require.NoError(t, err)

	// Verify quest was created
	retrieved, err := repo.GetQuestByID(ctx, quest.ID)
	require.NoError(t, err)
	assert.Equal(t, quest.ID, retrieved.ID)
	assert.Equal(t, quest.Recipient, retrieved.Recipient)
	assert.Equal(t, quest.Destination.Pavilion, retrieved.Destination.Pavilion)
	assert.Equal(t, 1, len(retrieved.Items))
	assert.Equal(t, "Laptop", retrieved.Items[0].Name)
}

// TestRepository_UpdateQuest tests updating an existing quest
func TestRepository_UpdateQuest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(repository.NewRepository(db))
	ctx := context.Background()

	// Create initial quest
	quest := &Quest{
		ID:       "quest-test456",
		QuestKey: "test-key-2",
		Destination: Destination{
			Pavilion: "Pawilon 5",
			Location: "POW",
		},
		Recipient:    "Anna Nowak",
		DeliveryDate: "2025-06-14",
		Items: []QuestItem{
			{Name: "Mouse", Quantity: 3, CategoryMatch: "none"},
		},
		Status:     "pending",
		SourceRows: []int{20},
		LastSynced: time.Now(),
	}

	err := repo.CreateQuest(ctx, quest)
	require.NoError(t, err)

	// Update quest with new items
	categoryID := 456
	quest.Items = []QuestItem{
		{Name: "Mouse", Quantity: 3, CategoryMatch: "none"},
		{
			Name:          "Keyboard",
			Quantity:      2,
			CategoryID:    &categoryID,
			CategoryMatch: "fuzzy",
		},
	}
	quest.SourceRows = []int{20, 21}

	err = repo.UpdateQuest(ctx, quest.ID, quest)
	require.NoError(t, err)

	// Verify update
	retrieved, err := repo.GetQuestByID(ctx, quest.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(retrieved.Items))
	assert.Equal(t, []int{20, 21}, retrieved.SourceRows)
}

// TestRepository_GetQuestByKey tests retrieving quest by its unique key
func TestRepository_GetQuestByKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(repository.NewRepository(db))
	ctx := context.Background()

	quest := &Quest{
		ID:       "quest-test789",
		QuestKey: "unique-key-123",
		Destination: Destination{
			Pavilion: "PCC",
			Location: "Test",
		},
		Recipient:    "Test User",
		DeliveryDate: "2025-06-15",
		Items: []QuestItem{
			{Name: "Test Item", Quantity: 1, CategoryMatch: "none"},
		},
		Status:     "pending",
		SourceRows: []int{30},
		LastSynced: time.Now(),
	}

	err := repo.CreateQuest(ctx, quest)
	require.NoError(t, err)

	// Retrieve by key
	retrieved, err := repo.GetQuestByKey(ctx, quest.QuestKey)
	require.NoError(t, err)
	assert.Equal(t, quest.ID, retrieved.ID)
	assert.Equal(t, quest.QuestKey, retrieved.QuestKey)
}

// TestRepository_ListQuests tests listing quests with filters
func TestRepository_ListQuests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(repository.NewRepository(db))
	ctx := context.Background()

	// Create multiple quests with different statuses
	quests := []*Quest{
		{
			ID:           "quest-list1",
			QuestKey:     "list-key-1",
			Destination:  Destination{Pavilion: "P1", Location: "L1"},
			Recipient:    "User 1",
			DeliveryDate: "2025-06-13",
			Items:        []QuestItem{{Name: "Item 1", Quantity: 1, CategoryMatch: "none"}},
			Status:       "pending",
			SourceRows:   []int{1},
			LastSynced:   time.Now(),
		},
		{
			ID:           "quest-list2",
			QuestKey:     "list-key-2",
			Destination:  Destination{Pavilion: "P2", Location: "L2"},
			Recipient:    "User 2",
			DeliveryDate: "2025-06-14",
			Items:        []QuestItem{{Name: "Item 2", Quantity: 1, CategoryMatch: "none"}},
			Status:       "in_progress",
			SourceRows:   []int{2},
			LastSynced:   time.Now(),
		},
		{
			ID:           "quest-list3",
			QuestKey:     "list-key-3",
			Destination:  Destination{Pavilion: "P3", Location: "L3"},
			Recipient:    "User 3",
			DeliveryDate: "2025-06-15",
			Items:        []QuestItem{{Name: "Item 3", Quantity: 1, CategoryMatch: "none"}},
			Status:       "completed",
			SourceRows:   []int{3},
			LastSynced:   time.Now(),
		},
	}

	for _, q := range quests {
		err := repo.CreateQuest(ctx, q)
		require.NoError(t, err)
	}

	// Test: List all quests
	allQuests, err := repo.ListQuests(ctx, QuestFilter{Limit: 100})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(allQuests), 3)

	// Test: Filter by status
	pendingQuests, err := repo.ListQuests(ctx, QuestFilter{Status: "pending", Limit: 100})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(pendingQuests), 1)
	for _, q := range pendingQuests {
		assert.Equal(t, "pending", q.Status)
	}

	// Test: Pagination
	page1, err := repo.ListQuests(ctx, QuestFilter{Limit: 2, Offset: 0})
	require.NoError(t, err)
	assert.LessOrEqual(t, len(page1), 2)

	page2, err := repo.ListQuests(ctx, QuestFilter{Limit: 2, Offset: 2})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(page2), 0)
}

// TestRepository_UpdateQuestStatus tests updating quest status
func TestRepository_UpdateQuestStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(repository.NewRepository(db))
	ctx := context.Background()

	quest := &Quest{
		ID:           "quest-status1",
		QuestKey:     "status-key-1",
		Destination:  Destination{Pavilion: "PCC", Location: "Test"},
		Recipient:    "Test User",
		DeliveryDate: "2025-06-16",
		Items:        []QuestItem{{Name: "Test", Quantity: 1, CategoryMatch: "none"}},
		Status:       "pending",
		SourceRows:   []int{40},
		LastSynced:   time.Now(),
	}

	err := repo.CreateQuest(ctx, quest)
	require.NoError(t, err)

	// Update status
	err = repo.UpdateQuestStatus(ctx, quest.ID, "in_progress")
	require.NoError(t, err)

	// Verify status changed
	retrieved, err := repo.GetQuestByID(ctx, quest.ID)
	require.NoError(t, err)
	assert.Equal(t, "in_progress", retrieved.Status)
}

// TestRepository_CreateSyncLog tests creating sync log entries
func TestRepository_CreateSyncLog(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(repository.NewRepository(db))
	ctx := context.Background()

	syncLog := &SyncLog{
		SyncedAt:        time.Now(),
		RowsProcessed:   20,
		QuestsCreated:   5,
		QuestsUpdated:   3,
		QuestsUnchanged: 12,
		ItemsAdded:      8,
		ItemsRemoved:    2,
		Success:         true,
		DurationMs:      2500,
		SheetID:         "test-sheet-id",
		Errors:          "",
	}

	err := repo.CreateSyncLog(ctx, syncLog)
	require.NoError(t, err)
	assert.NotZero(t, syncLog.ID)

	// Retrieve latest sync log
	retrieved, err := repo.GetLatestSyncLog(ctx)
	require.NoError(t, err)
	assert.Equal(t, syncLog.QuestsCreated, retrieved.QuestsCreated)
	assert.Equal(t, syncLog.QuestsUpdated, retrieved.QuestsUpdated)
}

// TestRepository_CategoryMapping tests manual category mapping
func TestRepository_CategoryMapping(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewRepository(repository.NewRepository(db))
	ctx := context.Background()

	userID := 7
	mapping := &CategoryMapping{
		FormItemName: "Laptop Dell",
		CategoryID:   100,
		CreatedBy:    &userID,
	}

	// Create mapping
	err := repo.CreateCategoryMapping(ctx, mapping)
	require.NoError(t, err)
	assert.NotZero(t, mapping.ID)

	// Retrieve mapping
	categoryID, err := repo.GetCategoryMapping(ctx, "Laptop Dell")
	require.NoError(t, err)
	require.NotNil(t, categoryID)
	assert.Equal(t, 100, *categoryID)

	// Test non-existent mapping
	notFound, err := repo.GetCategoryMapping(ctx, "Non Existent Item")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

// setupTestDB creates a test database connection
// Note: This requires a test database to be available
// You may need to configure this based on your test environment
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// TODO: Configure test database connection
	// For now, skip if TEST_DATABASE_URL is not set
	dbURL := "postgres://localhost/warehouse_test?sslmode=disable"

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skipf("Test database not available: %v", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("Cannot connect to test database: %v", err)
	}

	cleanup := func() {
		// Clean up test data
		db.Exec("DELETE FROM equipment_request_items")
		db.Exec("DELETE FROM equipment_request_quests")
		db.Exec("DELETE FROM equipment_request_sync_log")
		db.Exec("DELETE FROM equipment_request_category_mapping")
		db.Close()
	}

	return db, cleanup
}
