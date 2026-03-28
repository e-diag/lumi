package remnawave

import (
	"context"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/freeway-vpn/backend/internal/usecase"
)

// UsecaseAdapter адаптирует инфраструктурный Client под интерфейс usecase.RemnawaveClient.
type UsecaseAdapter struct {
	c *Client
}

func NewUsecaseAdapter(c *Client) *UsecaseAdapter {
	return &UsecaseAdapter{c: c}
}

func (a *UsecaseAdapter) CreateUser(ctx context.Context, userUUID, username string, tier domain.SubscriptionTier) error {
	return a.c.CreateUser(ctx, userUUID, username, tier)
}

func (a *UsecaseAdapter) DeleteUser(ctx context.Context, userUUID string) error {
	return a.c.DeleteUser(ctx, userUUID)
}

func (a *UsecaseAdapter) UpdateUserExpiry(ctx context.Context, userUUID string, expiresAt *time.Time) error {
	return a.c.UpdateUserExpiry(ctx, userUUID, expiresAt)
}

var _ usecase.RemnawaveClient = (*UsecaseAdapter)(nil)

