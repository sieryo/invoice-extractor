package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sieryo/invoice-extractor/internal/app/profile"
	"github.com/sieryo/invoice-extractor/internal/app/session"
	"github.com/sieryo/invoice-extractor/internal/profilepath"
	"golang.org/x/crypto/bcrypt"
)

type RegisterProfileInput struct {
	Name       string
	Alias      string
	CutoffDate int
	NPWP       string
	TKUID      string
}

type UpdateProfileInput struct {
	Name       string
	Alias      string
	CutoffDate int
	NPWP       string
	TKUID      string
}

type AuthService struct {
	profileRepo profile.Repository
	sessionRepo session.SessionRepository
	logger      *slog.Logger
	rootDir     string
}

func NewService(profileRepo profile.Repository, sessionRepo session.SessionRepository, logger *slog.Logger, rootDir string) *AuthService {
	return &AuthService{
		profileRepo: profileRepo,
		sessionRepo: sessionRepo,
		logger:      logger,
		rootDir:     rootDir,
	}
}

func (s *AuthService) Register(input RegisterProfileInput) error {
	name := strings.TrimSpace(input.Name)
	alias := strings.TrimSpace(strings.ToUpper(input.Alias))
	if name == "" {
		return errors.New("profile name is required")
	}
	if alias == "" {
		return errors.New("profile alias is required")
	}
	if input.CutoffDate <= 0 || input.CutoffDate > 31 {
		return errors.New("cutoff date must be between 1 and 31")
	}

	existingByName, err := s.profileRepo.GetByName(name)
	if existingByName != nil {
		s.logger.Warn("register failed: profile name taken", "name", name)
		return errors.New("profile name already taken")
	}
	if err != nil && err.Error() != "sql: no rows in result set" && err.Error() != "record not found" {
		s.logger.Error("register failed: db error checking profile name", "error", err, "name", name)
		return err
	}

	existingByAlias, err := s.profileRepo.GetByAlias(alias)
	if existingByAlias != nil {
		s.logger.Warn("register failed: profile alias taken", "alias", alias)
		return errors.New("profile alias already taken")
	}
	if err != nil && err.Error() != "sql: no rows in result set" && err.Error() != "record not found" {
		s.logger.Error("register failed: db error checking profile alias", "error", err, "alias", alias)
		return err
	}

	password := "expected"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("register failed: hashing error", "error", err)
		return err
	}

	p := &profile.Profile{
		ID:           uuid.New().String(),
		Name:         name,
		Alias:        alias,
		CutoffDate:   input.CutoffDate,
		NPWP:         strings.TrimSpace(input.NPWP),
		TKUID:        strings.TrimSpace(input.TKUID),
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
	}

	if err := s.profileRepo.Create(p); err != nil {
		s.logger.Error("register failed: db create error", "error", err)
		return err
	}

	if err := s.writeProfileMetadata(p); err != nil {
		s.logger.Error("register warning: failed to write profile metadata", "error", err, "profileId", p.ID)
	}

	s.logger.Info("profile registered successfully", "name", p.Name, "alias", p.Alias, "id", p.ID)
	return nil
}

func (s *AuthService) Login(name, password string) (string, error) {
	p, err := s.profileRepo.GetByName(strings.TrimSpace(name))
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(p.PasswordHash), []byte(password))
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	return s.createSession(p.ID)
}

func (s *AuthService) Logout(sessionID string) error {
	return s.sessionRepo.Delete(sessionID)
}

func (s *AuthService) GetSession(sessionID string) (*session.Session, error) {
	return s.sessionRepo.GetByID(sessionID)
}

func (s *AuthService) LoginByProfileID(profileID string) (string, error) {
	p, err := s.profileRepo.GetByID(profileID)
	if err != nil || p == nil {
		return "", errors.New("invalid credentials")
	}
	return s.createSession(p.ID)
}

func (s *AuthService) ListProfiles() ([]profile.Profile, error) {
	return s.profileRepo.List()
}

func (s *AuthService) GetProfileBySessionID(sessionID string) (*profile.Profile, error) {
	sess, err := s.sessionRepo.GetByID(sessionID)
	if err != nil {
		return nil, err
	}

	return s.profileRepo.GetByID(sess.ProfileID)
}

func (s *AuthService) UpdateProfile(profileID string, input UpdateProfileInput) (*profile.Profile, error) {
	current, err := s.profileRepo.GetByID(strings.TrimSpace(profileID))
	if err != nil || current == nil {
		return nil, errors.New("profile not found")
	}

	name := strings.TrimSpace(input.Name)
	alias := strings.TrimSpace(strings.ToUpper(input.Alias))
	if name == "" {
		return nil, errors.New("profile name is required")
	}
	if alias == "" {
		return nil, errors.New("profile alias is required")
	}
	if input.CutoffDate <= 0 || input.CutoffDate > 31 {
		return nil, errors.New("cutoff date must be between 1 and 31")
	}

	existingByName, err := s.profileRepo.GetByName(name)
	if err == nil && existingByName != nil && existingByName.ID != current.ID {
		return nil, errors.New("profile name already taken")
	}
	if err != nil && err.Error() != "sql: no rows in result set" && err.Error() != "record not found" {
		return nil, err
	}

	existingByAlias, err := s.profileRepo.GetByAlias(alias)
	if err == nil && existingByAlias != nil && existingByAlias.ID != current.ID {
		return nil, errors.New("profile alias already taken")
	}
	if err != nil && err.Error() != "sql: no rows in result set" && err.Error() != "record not found" {
		return nil, err
	}

	current.Name = name
	current.Alias = alias
	current.CutoffDate = input.CutoffDate
	current.NPWP = strings.TrimSpace(input.NPWP)
	current.TKUID = strings.TrimSpace(input.TKUID)

	if err := s.profileRepo.Update(current); err != nil {
		return nil, err
	}
	if err := s.writeProfileMetadata(current); err != nil {
		s.logger.Error("update profile warning: failed to write profile metadata", "error", err, "profileId", current.ID)
	}
	return current, nil
}

func (s *AuthService) GetLatestSessionByProfileID(profileID string) (*session.Session, error) {
	return s.sessionRepo.GetLatestByProfileID(strings.TrimSpace(profileID))
}

func (s *AuthService) createSession(profileID string) (string, error) {
	sessionID := uuid.New().String()
	sess := &session.Session{
		ID:        sessionID,
		ProfileID: profileID,
		ExpiresAt: time.Now().Add(24 * time.Hour * 30),
	}

	if err := s.sessionRepo.Create(sess); err != nil {
		return "", err
	}

	return sessionID, nil
}

func (s *AuthService) writeProfileMetadata(p *profile.Profile) error {
	if p == nil {
		return nil
	}
	profileDir := profilepath.ProfileDir(s.rootDir, p.ID)
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		return err
	}
	payload := map[string]any{
		"id":          p.ID,
		"name":        p.Name,
		"alias":       p.Alias,
		"cutoffDate":  p.CutoffDate,
		"npwp":        p.NPWP,
		"tkuId":       p.TKUID,
		"createdAt":   p.CreatedAt.UTC(),
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(profilepath.ProfileMetadataJSON(s.rootDir, p.ID), b, 0o644)
}
