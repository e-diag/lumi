package usecase_test

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/freeway-vpn/backend/internal/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Фейковые реализации репозиториев для тестов ─────────────────────────────

type fakeUserRepo struct {
	user *domain.User
}

func (r *fakeUserRepo) Create(_ context.Context, user *domain.User) error { return nil }
func (r *fakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	if r.user != nil && r.user.ID == id {
		return r.user, nil
	}
	return nil, domain.ErrUserNotFound
}
func (r *fakeUserRepo) GetByTelegramID(_ context.Context, _ int64) (*domain.User, error) {
	return r.user, nil
}
func (r *fakeUserRepo) GetBySubToken(_ context.Context, token string) (*domain.User, error) {
	if r.user != nil && r.user.SubToken == token {
		return r.user, nil
	}
	return nil, domain.ErrUserNotFound
}
func (r *fakeUserRepo) Count(_ context.Context) (int64, error) { return 1, nil }
func (r *fakeUserRepo) List(_ context.Context, _ string, _ int, _ int) ([]*domain.User, int64, error) {
	if r.user == nil {
		return nil, 0, nil
	}
	return []*domain.User{r.user}, 1, nil
}
func (r *fakeUserRepo) CountCreatedBetween(_ context.Context, _, _ time.Time) (int64, error) {
	return 0, nil
}
func (r *fakeUserRepo) Update(_ context.Context, _ *domain.User) error { return nil }
func (r *fakeUserRepo) Delete(_ context.Context, _ uuid.UUID) error    { return nil }

type fakeSubRepo struct {
	sub *domain.Subscription
}

func (r *fakeSubRepo) Create(_ context.Context, _ *domain.Subscription) error { return nil }
func (r *fakeSubRepo) GetByUserID(_ context.Context, _ uuid.UUID) (*domain.Subscription, error) {
	if r.sub == nil {
		return nil, domain.ErrSubscriptionNotFound
	}
	return r.sub, nil
}
func (r *fakeSubRepo) ListExpiredBefore(_ context.Context, _ time.Time) ([]*domain.Subscription, error) {
	return nil, nil
}
func (r *fakeSubRepo) ListExpiringBetween(_ context.Context, _, _ time.Time) ([]*domain.Subscription, error) {
	return nil, nil
}
func (r *fakeSubRepo) CountActive(_ context.Context, _ time.Time) (int64, error) { return 0, nil }
func (r *fakeSubRepo) CountActiveByTier(_ context.Context, _ domain.SubscriptionTier, _ time.Time) (int64, error) {
	return 0, nil
}
func (r *fakeSubRepo) CountExpired(_ context.Context, _ time.Time) (int64, error) { return 0, nil }
func (r *fakeSubRepo) Update(_ context.Context, _ *domain.Subscription) error     { return nil }
func (r *fakeSubRepo) Delete(_ context.Context, _ uuid.UUID) error                { return nil }

type fakeNodeRepo struct {
	nodes []*domain.Node
}

func (r *fakeNodeRepo) GetAll(_ context.Context) ([]*domain.Node, error) { return r.nodes, nil }
func (r *fakeNodeRepo) GetByRegion(_ context.Context, region domain.NodeRegion) ([]*domain.Node, error) {
	var result []*domain.Node
	for _, n := range r.nodes {
		if n.Region == region && n.Active {
			result = append(result, n)
		}
	}
	return result, nil
}

func (r *fakeNodeRepo) GetByRegionWithTopology(ctx context.Context, region domain.NodeRegion) ([]*domain.Node, error) {
	return r.GetByRegion(ctx, region)
}

func (r *fakeNodeRepo) ListActiveNodeDomains(_ context.Context) ([]*domain.NodeDomain, error) {
	return nil, nil
}

func (r *fakeNodeRepo) UpdateNodeDomain(_ context.Context, _ *domain.NodeDomain) error {
	return nil
}

func (r *fakeNodeRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Node, error) {
	for _, n := range r.nodes {
		if n.ID == id {
			return n, nil
		}
	}
	return nil, domain.ErrNodeNotFound
}
func (r *fakeNodeRepo) Create(_ context.Context, _ *domain.Node) error { return nil }
func (r *fakeNodeRepo) Update(_ context.Context, _ *domain.Node) error { return nil }

// ── Тесты ────────────────────────────────────────────────────────────────────

func makeTestNodes() []*domain.Node {
	return []*domain.Node{
		{
			ID:        uuid.New(),
			Name:      "EU-NL",
			Host:      "eu.example.com",
			Port:      443,
			Region:    domain.RegionEU,
			Transport: domain.TransportReality,
			PublicKey: "test-pub-key",
			ShortID:   "abcdef",
			SNI:       "www.google.com",
			Active:    true,
		},
		{
			ID:        uuid.New(),
			Name:      "CDN",
			Host:      "cdn.example.com",
			Port:      443,
			Region:    domain.RegionCDN,
			Transport: domain.TransportWS,
			SNI:       "cdn.example.com",
			WSPath:    "/ws",
			Active:    true,
		},
	}
}

func TestGenerateSubscription_FreeUserNoSub_OnlyEUNode(t *testing.T) {
	userID := uuid.New()
	user := &domain.User{ID: userID, TelegramID: 1, SubToken: "token-1"}

	uc := usecase.NewConfigUseCase(
		&fakeUserRepo{user: user},
		&fakeSubRepo{sub: nil}, // нет подписки → Free
		&fakeNodeRepo{nodes: makeTestNodes()},
		"",
		"",
	)

	encoded, err := uc.GenerateSubscription(context.Background(), userID)
	require.NoError(t, err)
	require.NotEmpty(t, encoded)

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)

	lines := strings.Split(string(decoded), "\n")
	assert.Len(t, lines, 1, "Free tier: только EU нода")
	assert.True(t, strings.HasPrefix(lines[0], "vless://"))
	assert.Contains(t, lines[0], "eu.example.com")
	assert.Contains(t, lines[0], "reality")
}

func TestGenerateSubscription_PremiumUser_AllNodes(t *testing.T) {
	userID := uuid.New()
	user := &domain.User{ID: userID, TelegramID: 2, SubToken: "token-2"}
	sub := &domain.Subscription{
		ID:        uuid.New(),
		UserID:    userID,
		Tier:      domain.TierPremium,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	nodes := []*domain.Node{
		{ID: uuid.New(), Name: "EU", Host: "eu.example.com", Port: 443, Region: domain.RegionEU, Transport: domain.TransportReality, PublicKey: "pk", ShortID: "sid", SNI: "google.com", Active: true},
		{ID: uuid.New(), Name: "USA", Host: "usa.example.com", Port: 443, Region: domain.RegionUSA, Transport: domain.TransportReality, PublicKey: "pk2", ShortID: "sid2", SNI: "google.com", Active: true},
		{ID: uuid.New(), Name: "CDN", Host: "cdn.example.com", Port: 443, Region: domain.RegionCDN, Transport: domain.TransportWS, SNI: "cdn.example.com", WSPath: "/ws", Active: true},
	}

	uc := usecase.NewConfigUseCase(
		&fakeUserRepo{user: user},
		&fakeSubRepo{sub: sub},
		&fakeNodeRepo{nodes: nodes},
		"",
		"",
	)

	encoded, err := uc.GenerateSubscription(context.Background(), userID)
	require.NoError(t, err)

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)

	lines := strings.Split(string(decoded), "\n")
	assert.Len(t, lines, 3, "Premium: все 3 ноды")
}

func TestGenerateSubscription_CDNNodeIsLast(t *testing.T) {
	userID := uuid.New()
	user := &domain.User{ID: userID, TelegramID: 4, SubToken: "token-4"}
	sub := &domain.Subscription{
		ID:        uuid.New(),
		UserID:    userID,
		Tier:      domain.TierPremium,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	nodes := []*domain.Node{
		{ID: uuid.New(), Name: "USA", Host: "usa.example.com", Port: 443, Region: domain.RegionUSA, Transport: domain.TransportReality, PublicKey: "pk2", ShortID: "sid2", SNI: "google.com", Active: true},
		{
			ID:              uuid.New(),
			Name:            "CDN-Yandex",
			Host:            "vpn.freeway.app",
			Port:            443,
			Region:          domain.RegionCDN,
			Transport:       domain.TransportGRPC,
			GRPCServiceName: "vless",
			Active:          true,
		},
		{ID: uuid.New(), Name: "EU", Host: "eu.example.com", Port: 443, Region: domain.RegionEU, Transport: domain.TransportReality, PublicKey: "pk", ShortID: "sid", SNI: "google.com", Active: true},
	}

	uc := usecase.NewConfigUseCase(&fakeUserRepo{user: user}, &fakeSubRepo{sub: sub}, &fakeNodeRepo{nodes: nodes}, "", "")
	encoded, err := uc.GenerateSubscription(context.Background(), userID)
	require.NoError(t, err)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)
	lines := strings.Split(string(decoded), "\n")
	require.GreaterOrEqual(t, len(lines), 1)
	last := lines[len(lines)-1]
	assert.Contains(t, last, "type=grpc")
	assert.Contains(t, last, "serviceName=vless")
}

func TestGenerateSubscription_NoActiveNodes_ReturnsError(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	user := &domain.User{ID: userID, TelegramID: 9, SubToken: "token-9"}
	uc := usecase.NewConfigUseCase(
		&fakeUserRepo{user: user},
		&fakeSubRepo{sub: nil},
		&fakeNodeRepo{nodes: nil},
		"",
		"",
	)
	_, err := uc.GenerateSubscription(context.Background(), userID)
	require.Error(t, err)
}

func TestGenerateSubscription_ExpiredSub_FallbackToFree(t *testing.T) {
	userID := uuid.New()
	user := &domain.User{ID: userID, TelegramID: 3, SubToken: "token-3"}
	sub := &domain.Subscription{
		ID:        uuid.New(),
		UserID:    userID,
		Tier:      domain.TierBasic,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // истекла
	}

	uc := usecase.NewConfigUseCase(
		&fakeUserRepo{user: user},
		&fakeSubRepo{sub: sub},
		&fakeNodeRepo{nodes: makeTestNodes()},
		"",
		"",
	)

	encoded, err := uc.GenerateSubscription(context.Background(), userID)
	require.NoError(t, err)

	decoded, _ := base64.StdEncoding.DecodeString(encoded)
	lines := strings.Split(string(decoded), "\n")
	assert.Len(t, lines, 1, "истекшая подписка → Free (только EU)")
}
