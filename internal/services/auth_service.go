package services

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sieryo/invoice-extractor/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	Create(u *models.User) error
	GetByUsername(username string) (*models.User, error)
	GetByID(id string) (*models.User, error)
}

type SessionRepository interface {
	Create(s *models.Session) error
	GetByID(id string) (*models.Session, error)
	Delete(id string) error
}

type AuthService struct {
	userRepo    UserRepository
	sessionRepo SessionRepository
}

func NewAuth(userRepo UserRepository, sessionRepo SessionRepository) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

func (s *AuthService) Register(username, password string) error {
	existingUser, err := s.userRepo.GetByUsername(username)
	if existingUser != nil {
		return errors.New("username already taken")
	}
	if err != nil && err.Error() != "sql: no rows in result set" && err.Error() != "record not found" {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &models.User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
	}

	return s.userRepo.Create(user)
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
	session := &models.Session{
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
