package auth_test

import (
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/sieryo/invoice-extractor/internal/app/auth"
	"github.com/sieryo/invoice-extractor/internal/app/session"
	"github.com/sieryo/invoice-extractor/internal/app/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(u *user.User) error {
	args := m.Called(u)
	return args.Error(0)
}

func (m *MockUserRepository) GetByUsername(username string) (*user.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) GetByID(id string) (*user.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) List() ([]user.User, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]user.User), args.Error(1)
}

type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Create(s *session.Session) error {
	args := m.Called(s)
	return args.Error(0)
}

func (m *MockSessionRepository) GetByID(id string) (*session.Session, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*session.Session), args.Error(1)
}

func (m *MockSessionRepository) Delete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func TestRegister(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockSessionRepo := new(MockSessionRepository)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	authService := auth.NewService(mockUserRepo, mockSessionRepo, logger)

	t.Run("Success", func(t *testing.T) {
		username := "testuser"
		password := "password123"

		mockUserRepo.On("GetByUsername", username).Return(nil, errors.New("record not found"))
		mockUserRepo.On("Create", mock.AnythingOfType("*user.User")).Return(nil)

		err := authService.Register(username, password)
		assert.NoError(t, err)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("UsernameTaken", func(t *testing.T) {
		username := "existinguser"
		password := "password123"

		existingUser := &user.User{Username: username}
		mockUserRepo.On("GetByUsername", username).Return(existingUser, nil)

		err := authService.Register(username, password)
		assert.Error(t, err)
		assert.Equal(t, "username already taken", err.Error())
		mockUserRepo.AssertExpectations(t)
	})
}

func TestLogin(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockSessionRepo := new(MockSessionRepository)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	authService := auth.NewService(mockUserRepo, mockSessionRepo, logger)

	t.Run("Success", func(t *testing.T) {
		username := "testuser"
		password := "password123"
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

		user := &user.User{
			ID:           uuid.New().String(),
			Username:     username,
			PasswordHash: string(hashedPassword),
		}

		mockUserRepo.On("GetByUsername", username).Return(user, nil)
		mockSessionRepo.On("Create", mock.AnythingOfType("*user.Session")).Return(nil)

		sessionID, err := authService.Login(username, password)
		assert.NoError(t, err)
		assert.NotEmpty(t, sessionID)
		mockUserRepo.AssertExpectations(t)
		mockSessionRepo.AssertExpectations(t)
	})

	t.Run("InvalidCredentials_WrongPassword", func(t *testing.T) {
		username := "testuser"
		password := "wrongpassword"
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)

		user := &user.User{
			Username:     username,
			PasswordHash: string(hashedPassword),
		}

		mockUserRepo.On("GetByUsername", username).Return(user, nil)

		_, err := authService.Login(username, password)
		assert.Error(t, err)
		assert.Equal(t, "invalid credentials", err.Error())
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("InvalidCredentials_UserNotFound", func(t *testing.T) {
		username := "nonexistent"
		password := "password123"

		mockUserRepo.On("GetByUsername", username).Return(nil, errors.New("record not found"))

		_, err := authService.Login(username, password)
		assert.Error(t, err)
		assert.Equal(t, "invalid credentials", err.Error())
		mockUserRepo.AssertExpectations(t)
	})
}

func TestLogout(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockSessionRepo := new(MockSessionRepository)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	authService := auth.NewService(mockUserRepo, mockSessionRepo, logger)

	t.Run("Success", func(t *testing.T) {
		sessionID := "session-id-123"
		mockSessionRepo.On("Delete", sessionID).Return(nil)

		err := authService.Logout(sessionID)
		assert.NoError(t, err)
		mockSessionRepo.AssertExpectations(t)
	})
}
