package collection

import (
	"context"
	"time"
)

type Repository interface {
	Create(ctx context.Context, c *Collection) error
	FindByID(ctx context.Context, id string) (*Collection, error)

	ListByUserID(ctx context.Context, userID string) ([]*Collection, error)
	ListChildren(ctx context.Context, userID string, parentID *string) ([]*Collection, error)

	UpdatePhase(ctx context.Context, id string, phase Phase) error
	UpdateSummary(ctx context.Context, id string, total, ready, warning, failed, duplicate int) error
	Restore(ctx context.Context, id string) error

	// Legacy compatibility methods.
	UpdateStatus(ctx context.Context, id string, status Status) error
	UpdateName(ctx context.Context, id string, name string) error
	Freeze(ctx context.Context, id string, frozenBy string, frozenAt time.Time) error
	Delete(ctx context.Context, id string) error
}
