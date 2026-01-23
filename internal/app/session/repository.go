package session

type SessionRepository interface {
	Create(s *Session) error
	GetByID(id string) (*Session, error)
	Delete(id string) error
}
