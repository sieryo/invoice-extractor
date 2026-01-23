package service

import (
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/sieryo/invoice-extractor/internal/model"
	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	Create(u *model.User) error
	GetByUsername(username string) (*model.User, error)
	GetByID(id string) (*model.User, error)
}

type SessionRepository interface {
	Create(s *model.Session) error
	GetByID(id string) (*model.Session, error)
	Delete(id string) error
}

type AuthService struct {
	userRepo    UserRepository
	sessionRepo SessionRepository
	logger      *slog.Logger
}

func NewAuth(userRepo UserRepository, sessionRepo SessionRepository, logger *slog.Logger) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		logger:      logger,
	}
}

func (s *AuthService) Register(username, password string) error {
	existingUser, err := s.userRepo.GetByUsername(username)
	if existingUser != nil {
		s.logger.Warn("register failed: username taken", "username", username)
		return errors.New("username already taken")
	}
	if err != nil && err.Error() != "sql: no rows in result set" && err.Error() != "record not found" {
		s.logger.Error("register failed: db error checking username", "error", err, "username", username)
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("register failed: hashing error", "error", err)
		return err
	}

	user := &model.User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(user); err != nil {
		s.logger.Error("register failed: db create error", "error", err)
		return err
	}

	s.logger.Info("user registered successfully", "username", username, "id", user.ID)
	return nil
}

func (s *AuthService) Login(username, password string) (string, error) {
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	sessionID := uuid.New().String()
	session := &model.Session{
		ID:        sessionID,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour * 30),
	}

	if err := s.sessionRepo.Create(session); err != nil {
		return "", err
	}

	return sessionID, nil
}

func (s *AuthService) Logout(sessionID string) error {
	return s.sessionRepo.Delete(sessionID)
}
