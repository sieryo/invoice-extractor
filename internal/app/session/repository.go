package session

// TODO: Nanti tambahin ctx.Context
type SessionRepository interface {
	Create(s *Session) error
	GetByID(id string) (*Session, error)
	Delete(id string) error
}
