package collection

import "time"

type Status string

const (
	StatusActive    Status = "active"
	StatusCommitted Status = "committed"
	StatusExpired   Status = "expired"
)

type Collection struct {
	ID string `json:"id"`

	UserID string `json:"user_id"`
	Status Status `json:"status"`
	Name   string `json:"name"`

	CreatedAt time.Time  `json:"created_at"`
	ExpiredAt *time.Time `json:"expired_at,omitempty"`
}

func NewCollection(id string, userID string, name string, now time.Time) *Collection {
	return &Collection{
		ID:        id,
		UserID:    userID,
		Name:      name,
		Status:    StatusActive,
		CreatedAt: now,
	}
}

func (c *Collection) IsActive() bool {
	return c.Status == StatusActive
}

func (c *Collection) IsCommitted() bool {
	return c.Status == StatusCommitted
}

func (c *Collection) IsExpired() bool {
	return c.Status == StatusExpired
}

// Commit menandai collection sudah final dan tidak bisa menerima file baru
func (c *Collection) Commit() error {
	if c.Status != StatusActive {
		return ErrInvalidStatusTransition
	}

	c.Status = StatusCommitted
	return nil
}

// Expire menandai collection sudah kadaluarsa
func (c *Collection) Expire(at time.Time) error {
	if c.Status == StatusExpired {
		return nil
	}

	c.Status = StatusExpired
	c.ExpiredAt = &at
	return nil
}
