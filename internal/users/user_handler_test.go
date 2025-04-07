package users

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"warehouse/pkg/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) PersistUser(req models.CreateUserRequest, hashedPassword []byte) error {
	args := m.Called(req, hashedPassword)
	return args.Error(0)
}

func (m *MockUserRepository) GetUser(id int) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetUsers() ([]models.User, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.User), args.Error(1)
}

func setupTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", "1")
	c.Set("role", "admin")
	return c, w
}

func TestRegisterUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockRepo := new(MockUserRepository)
	handler := NewHandler(mockRepo)

	tests := []struct {
		name           string
		payload        models.CreateUserRequest
		setupMock      func()
		expectedStatus int
	}{
		{
			name: "successful registration",
			payload: models.CreateUserRequest{
				Username: "testuser",
				Password: "password123",
				Fullname: "Test User",
				Role:     "user",
			},
			setupMock: func() {
				mockRepo.On("PersistUser", mock.Anything, mock.Anything).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "repository error",
			payload: models.CreateUserRequest{
				Username: "testuser",
				Password: "password123",
				Fullname: "Test User",
				Role:     "user",
			},
			setupMock: func() {
				mockRepo.On("PersistUser", mock.Anything, mock.Anything).Return(errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.ExpectedCalls = nil
			tt.setupMock()
			c, w := setupTestContext()

			body, _ := json.Marshal(tt.payload)
			c.Request = httptest.NewRequest("POST", "/users", bytes.NewBuffer(body))

			handler.RegisterUser(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUpdateUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockRepo := new(MockUserRepository)
	handler := NewHandler(mockRepo)

	tests := []struct {
		name           string
		userID         string
		payload        models.UpdateUserRequest
		setupMock      func()
		expectedStatus int
	}{
		{
			name:   "successful update",
			userID: "1",
			payload: models.UpdateUserRequest{
				Fullname: stringPtr("Updated Name"),
				Role:     stringPtr("admin"),
			},
			setupMock: func() {
				mockRepo.On("GetUser", 1).Return(&models.User{
					ID:       1,
					Username: "testuser",
					Role:     "user",
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "user not found",
			userID: "999",
			payload: models.UpdateUserRequest{
				Fullname: stringPtr("Updated Name"),
			},
			setupMock: func() {
				mockRepo.On("GetUser", 999).Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.ExpectedCalls = nil
			tt.setupMock()
			c, w := setupTestContext()

			body, _ := json.Marshal(tt.payload)
			c.Request = httptest.NewRequest("PATCH", "/users/"+tt.userID, bytes.NewBuffer(body))
			c.Params = []gin.Param{{Key: "id", Value: tt.userID}}

			handler.UpdateUser(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetUserList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockRepo := new(MockUserRepository)
	handler := NewHandler(mockRepo)

	tests := []struct {
		name           string
		setupMock      func()
		expectedStatus int
	}{
		{
			name: "successful list retrieval",
			setupMock: func() {
				mockRepo.On("GetUsers").Return([]models.User{
					{ID: 1, Username: "user1"},
					{ID: 2, Username: "user2"},
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "repository error",
			setupMock: func() {
				mockRepo.On("GetUsers").Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.ExpectedCalls = nil
			tt.setupMock()
			c, w := setupTestContext()
			c.Request = httptest.NewRequest("GET", "/users", nil)

			handler.GetUserList(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
