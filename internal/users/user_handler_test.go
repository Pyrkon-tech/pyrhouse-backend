package users

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"warehouse/pkg/models"
	"warehouse/pkg/roles"

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

func (m *MockUserRepository) AddUserPoints(id int, points int) error {
	args := m.Called(id, points)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateUser(id int, changes *models.UserChanges) error {
	args := m.Called(id, changes)
	return args.Error(0)
}

func (m *MockUserRepository) DeleteUser(id int) error {
	args := m.Called(id)
	return args.Error(0)
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
				Role:     rolesPtr(roles.Admin),
			},
			setupMock: func() {
				// Mock dla GetUser
				mockRepo.On("GetUser", 1).Return(&models.User{
					ID:       1,
					Username: "testuser",
					Role:     "user",
				}, nil)

				// Mock dla UpdateUser
				mockRepo.On("UpdateUser", 1, mock.MatchedBy(func(changes *models.UserChanges) bool {
					return changes.Role != nil && *changes.Role == string(roles.Admin)
				})).Return(nil)

				// Mock dla drugiego GetUser po aktualizacji
				mockRepo.On("GetUser", 1).Return(&models.User{
					ID:       1,
					Username: "testuser",
					Role:     roles.Admin,
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

func TestAddUserPoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockRepo := new(MockUserRepository)
	handler := NewHandler(mockRepo)

	tests := []struct {
		name    string
		userID  string
		payload struct {
			Points int `json:"points"`
		}
		setupMock      func()
		expectedStatus int
	}{
		{
			name:   "successful points addition",
			userID: "1",
			payload: struct {
				Points int `json:"points"`
			}{
				Points: 10,
			},
			setupMock: func() {
				mockRepo.On("AddUserPoints", 1, 10).Return(nil)
				mockRepo.On("GetUser", 1).Return(&models.User{
					ID:       1,
					Username: "testuser",
					Points:   10,
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "successful points subtraction",
			userID: "1",
			payload: struct {
				Points int `json:"points"`
			}{
				Points: -5,
			},
			setupMock: func() {
				mockRepo.On("AddUserPoints", 1, -5).Return(nil)
				mockRepo.On("GetUser", 1).Return(&models.User{
					ID:       1,
					Username: "testuser",
					Points:   5,
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "invalid user ID",
			userID: "invalid",
			payload: struct {
				Points int `json:"points"`
			}{
				Points: 10,
			},
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "repository error",
			userID: "1",
			payload: struct {
				Points int `json:"points"`
			}{
				Points: 10,
			},
			setupMock: func() {
				mockRepo.On("AddUserPoints", 1, 10).Return(errors.New("db error"))
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
			c.Request = httptest.NewRequest("POST", "/users/"+tt.userID+"/points", bytes.NewBuffer(body))
			c.Params = []gin.Param{{Key: "id", Value: tt.userID}}

			handler.AddUserPoints(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUpdateUserPassword(t *testing.T) {
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
			name:   "successful password update",
			userID: "1",
			payload: models.UpdateUserRequest{
				Password: stringPtr("newPassword123"),
			},
			setupMock: func() {
				// Mock dla GetUser
				mockRepo.On("GetUser", 1).Return(&models.User{
					ID:           1,
					Username:     "testuser",
					PasswordHash: "oldHash",
					Role:         "user",
				}, nil)

				// Mock dla UpdateUser
				mockRepo.On("UpdateUser", 1, mock.MatchedBy(func(changes *models.UserChanges) bool {
					return changes.PasswordHash != nil && *changes.PasswordHash != "oldHash"
				})).Return(nil)

				// Mock dla drugiego GetUser po aktualizacji
				mockRepo.On("GetUser", 1).Return(&models.User{
					ID:           1,
					Username:     "testuser",
					PasswordHash: "newHash",
					Role:         "user",
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "password too short",
			userID: "1",
			payload: models.UpdateUserRequest{
				Password: stringPtr("123"),
			},
			setupMock: func() {
				// Mock dla GetUser
				mockRepo.On("GetUser", 1).Return(&models.User{
					ID:           1,
					Username:     "testuser",
					PasswordHash: "oldHash",
					Role:         "user",
				}, nil)

				// Nie powinno być wywołania UpdateUser, ponieważ hasło jest za krótkie
				// Nie dodajemy mocka dla UpdateUser
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "user not found",
			userID: "999",
			payload: models.UpdateUserRequest{
				Password: stringPtr("newPassword123"),
			},
			setupMock: func() {
				// Mock dla GetUser zwracający błąd
				mockRepo.On("GetUser", 999).Return(nil, errors.New("user not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "repository error on update",
			userID: "1",
			payload: models.UpdateUserRequest{
				Password: stringPtr("newPassword123"),
			},
			setupMock: func() {
				// Mock dla GetUser
				mockRepo.On("GetUser", 1).Return(&models.User{
					ID:           1,
					Username:     "testuser",
					PasswordHash: "oldHash",
					Role:         "user",
				}, nil)

				// Mock dla UpdateUser zwracający błąd
				mockRepo.On("UpdateUser", 1, mock.Anything).Return(errors.New("database error"))
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
			c.Request = httptest.NewRequest("PATCH", "/users/"+tt.userID, bytes.NewBuffer(body))
			c.Params = []gin.Param{{Key: "id", Value: tt.userID}}

			handler.UpdateUser(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestDeleteUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockRepo := new(MockUserRepository)
	handler := NewHandler(mockRepo)

	tests := []struct {
		name           string
		userID         string
		setupMock      func()
		expectedStatus int
	}{
		{
			name:   "successful deletion",
			userID: "1",
			setupMock: func() {
				mockRepo.On("DeleteUser", 1).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "user has transfers",
			userID: "1",
			setupMock: func() {
				mockRepo.On("DeleteUser", 1).Return(fmt.Errorf("nie można usunąć użytkownika, ponieważ ma przypisane transfery"))
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "invalid user ID",
			userID:         "invalid",
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "user not found",
			userID: "999",
			setupMock: func() {
				mockRepo.On("DeleteUser", 999).Return(fmt.Errorf("nie znaleziono użytkownika o id: 999"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "unauthorized access",
			userID: "2",
			setupMock: func() {
				// Nie dodajemy mocka, bo handler zakończy się wcześniej
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.ExpectedCalls = nil
			tt.setupMock()
			c, w := setupTestContext()

			c.Request = httptest.NewRequest("DELETE", "/users/"+tt.userID, nil)
			c.Params = []gin.Param{{Key: "id", Value: tt.userID}}

			handler.DeleteUser(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

func rolesPtr(r roles.Role) *roles.Role {
	return &r
}
