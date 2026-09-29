package equipment_requests

import (
	"context"
	"database/sql"
	"testing"

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
				Name:        "Laptop",
				Quantity:    intPtr(2),
				CategoryID:  &categoryID,
				BudgetOwner: "Anna Nowak",
				Notes:       "Test note",
			},
		},
		Status: "pending",
		Source: SourceSheet,
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
			{Name: "Mouse", Quantity: intPtr(3)},
		},
		Status: "pending",
		Source: SourceSheet,
	}

	err := repo.CreateQuest(ctx, quest)
	require.NoError(t, err)

	// Update quest with new items
	categoryID := 456
	quest.Items = []QuestItem{
		{Name: "Mouse", Quantity: intPtr(3)},
		{
			Name:       "Keyboard",
			Quantity:   intPtr(2),
			CategoryID: &categoryID,
		},
	}

	err = repo.UpdateQuest(ctx, quest.ID, quest)
	require.NoError(t, err)

	// Verify update
	retrieved, err := repo.GetQuestByID(ctx, quest.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(retrieved.Items))
	assert.Equal(t, SourceSheet, retrieved.Source)
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
			{Name: "Test Item", Quantity: intPtr(1)},
		},
		Status: "pending",
		Source: SourceSheet,
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
			Items:        []QuestItem{{Name: "Item 1", Quantity: intPtr(1)}},
			Status:       "pending",
			Source:       SourceSheet,
		},
		{
			ID:           "quest-list2",
			QuestKey:     "list-key-2",
			Destination:  Destination{Pavilion: "P2", Location: "L2"},
			Recipient:    "User 2",
			DeliveryDate: "2025-06-14",
			Items:        []QuestItem{{Name: "Item 2", Quantity: intPtr(1)}},
			Status:       "in_progress",
			Source:       SourceSheet,
		},
		{
			ID:           "quest-list3",
			QuestKey:     "list-key-3",
			Destination:  Destination{Pavilion: "P3", Location: "L3"},
			Recipient:    "User 3",
			DeliveryDate: "2025-06-15",
			Items:        []QuestItem{{Name: "Item 3", Quantity: intPtr(1)}},
			Status:       "completed",
			Source:       SourceSheet,
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
		Items:        []QuestItem{{Name: "Test", Quantity: intPtr(1)}},
		Status:       "pending",
		Source:       SourceSheet,
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
		db.Close()
	}

	return db, cleanup
}
