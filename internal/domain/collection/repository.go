package collection

import (
	"context"
	"time"
)

type Repository interface {
	Create(ctx context.Context, c *Collection) error
	FindByID(ctx context.Context, id string) (*Collection, error)

	ListByUserID(ctx context.Context, userID string) ([]*Collection, error)

	UpdateStatus(ctx context.Context, id string, status Status) error
	Delete(ctx context.Context, id string) error
	Expire(ctx context.Context, now time.Time) error
}
