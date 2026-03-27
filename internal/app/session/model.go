package session

import "time"

type Session struct {
	ID        string
	ProfileID string
	ExpiresAt time.Time
}
