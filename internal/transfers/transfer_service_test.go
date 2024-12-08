package transfers

import (
	"errors"
	"testing"
	"warehouse/pkg/models"

	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

type MockTransferRepository struct {
	mock.Mock
}

func (m *MockTransferRepository) InsertTransferRecord(tx *goqu.TxDatabase, req models.TransferRequest) (int, error) {
	args := m.Called(tx, req)
	return args.Int(0), args.Error(1)
}

func (m *MockTransferRepository) InsertSerializedItemTransferRecord(tx *goqu.TxDatabase, transferID int, assets []int) error {
	args := m.Called(tx, transferID, assets)
	return args.Error(0)
}

func (m *MockTransferRepository) MoveSerializedItems(tx *goqu.TxDatabase, assets []int, locationID int, status string) error {
	args := m.Called(tx, assets, locationID, status)
	return args.Error(0)
}

type MockStockRepository struct {
	mock.Mock
}

func (m *MockStockRepository) MoveNonSerializedItems(tx *goqu.TxDatabase, assets []models.UnserializedItemRequest, locationID, fromLocationID int) error {
	args := m.Called(tx, assets, locationID, fromLocationID)
	return args.Error(0)
}

func TestPerformTransfer(t *testing.T) {
	mockRepo := new(MockRepository)
	mockTransferRepo := new(MockTransferRepository)
	mockStockRepo := new(MockStockRepository)

	transferService := TransferService{
		r:  mockRepo,
		tr: mockTransferRepo,
	}

	tx := new(goqu.TxDatabase) // Mock the transaction if needed

	// Sample input for a transfer request
	req := models.TransferRequest{
		LocationID:               1,
		FromLocationID:           2,
		SerialziedItemCollection: []int{101, 102},
		UnserializedItemCollection: []models.UnserializedItemRequest{
			{ItemCategoryID: 1, Quantity: 10},
		},
	}

	transitStatus := "in_transit"

	// Mocking successful behavior
	mockTransferRepo.On("InsertTransferRecord", tx, req).Return(123, nil).Once()
	mockTransferRepo.On("InsertSerializedItemTransferRecord", tx, 123, req.SerialziedItemCollection).Return(nil).Once()
	mockTransferRepo.On("MoveSerializedItems", tx, req.SerialziedItemCollection, req.LocationID, transitStatus).Return(nil).Once()
	mockStockRepo.On("MoveNonSerializedItems", tx, req.UnserializedItemCollection, req.LocationID, req.FromLocationID).Return(nil).Once()

	transferID, err := transferService.PerformTransfer(req, transitStatus)

	assert.NoError(t, err)
	assert.Equal(t, 123, transferID)

	// Mocking failure scenario
	mockTransferRepo.On("InsertTransferRecord", tx, req).Return(0, errors.New("failed to insert transfer record")).Once()

	_, err = transferService.PerformTransfer(req, transitStatus)
	assert.Error(t, err)

	mockTransferRepo.AssertExpectations(t)
	mockStockRepo.AssertExpectations(t)
}
