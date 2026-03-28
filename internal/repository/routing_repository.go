package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type routingRepository struct {
	db *gorm.DB
}

// NewRoutingRepository создаёт реализацию RoutingRepository на основе GORM.
func NewRoutingRepository(db *gorm.DB) RoutingRepository {
	return &routingRepository{db: db}
}

func (r *routingRepository) GetAll(ctx context.Context) ([]*domain.RoutingRule, error) {
	var rules []*domain.RoutingRule
	if err := r.db.WithContext(ctx).Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("repository: routing get all: %w", err)
	}
	return rules, nil
}

func (r *routingRepository) GetActive(ctx context.Context) ([]*domain.RoutingRule, error) {
	var rules []*domain.RoutingRule
	if err := r.db.WithContext(ctx).Where("active = true").Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("repository: routing get active: %w", err)
	}
	return rules, nil
}

func (r *routingRepository) SaveDomains(ctx context.Context, source string, action domain.RouteAction, domains []string) error {
	if len(domains) == 0 {
		return nil
	}
	now := time.Now()
	rules := make([]domain.RoutingRule, 0, len(domains))
	seen := map[string]struct{}{}
	for _, d := range domains {
		d = normalizeDomain(d)
		if d == "" {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		typ := domain.RuleTypeDomain
		if strings.Contains(d, "/") {
			typ = domain.RuleTypeCIDR
		}
		rules = append(rules, domain.RoutingRule{
			ID:        uuid.New(),
			Type:      typ,
			Value:     d,
			Source:    source,
			Action:    action,
			IsManual:  false,
			Active:    true,
			UpdatedAt: now,
		})
	}
	if len(rules) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "value"}},
		DoUpdates: clause.Assignments(map[string]any{
			"source":    source,
			"action":    action,
			"active":    true,
			"updated_at": now,
		}),
	}).Create(&rules).Error; err != nil {
		return fmt.Errorf("repository: routing save domains: %w", err)
	}
	return nil
}

func (r *routingRepository) GetRoutingList(ctx context.Context) (*domain.RoutingList, error) {
	var rules []*domain.RoutingRule
	if err := r.db.WithContext(ctx).Where("active = true").Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("repository: routing get list: %w", err)
	}
	out := &domain.RoutingList{
		ProxyEU:  []string{},
		ProxyUSA: []string{},
		Direct:   []string{},
		Manual:   []string{},
	}
	var latest time.Time
	for _, rr := range rules {
		if rr.UpdatedAt.After(latest) {
			latest = rr.UpdatedAt
		}
		switch rr.Action {
		case domain.ActionProxyEU:
			out.ProxyEU = append(out.ProxyEU, rr.Value)
		case domain.ActionProxyUSA:
			out.ProxyUSA = append(out.ProxyUSA, rr.Value)
		case domain.ActionDirect:
			out.Direct = append(out.Direct, rr.Value)
		}
		if rr.IsManual {
			out.Manual = append(out.Manual, rr.Value)
		}
	}
	sort.Strings(out.ProxyEU)
	sort.Strings(out.ProxyUSA)
	sort.Strings(out.Direct)
	sort.Strings(out.Manual)
	out.UpdatedAt = latest
	if latest.IsZero() {
		out.Version = ""
	} else {
		out.Version = latest.UTC().Format("2006-01-02")
	}
	return out, nil
}

func (r *routingRepository) GetVersion(ctx context.Context) (string, error) {
	var rr domain.RoutingRule
	err := r.db.WithContext(ctx).Order("updated_at DESC").First(&rr).Error
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("repository: routing get version: %w", err)
	}
	return rr.UpdatedAt.UTC().Format("2006-01-02"), nil
}

func (r *routingRepository) AddManualDomain(ctx context.Context, value string, action domain.RouteAction) error {
	value = normalizeDomain(value)
	if value == "" {
		return nil
	}
	typ := domain.RuleTypeDomain
	if strings.Contains(value, "/") {
		typ = domain.RuleTypeCIDR
	}
	rule := &domain.RoutingRule{
		ID:       uuid.New(),
		Type:     typ,
		Value:    value,
		Source:   "manual",
		Action:   action,
		IsManual: true,
		Active:   true,
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "value"}},
		DoUpdates: clause.AssignmentColumns([]string{"source", "action", "is_manual", "active", "updated_at"}),
	}).Create(rule).Error; err != nil {
		return fmt.Errorf("repository: routing add manual domain: %w", err)
	}
	return nil
}

func (r *routingRepository) DeleteManualDomain(ctx context.Context, value string) error {
	value = normalizeDomain(value)
	if err := r.db.WithContext(ctx).
		Where("value = ? AND is_manual = true", value).
		Delete(&domain.RoutingRule{}).Error; err != nil {
		return fmt.Errorf("repository: routing delete manual domain: %w", err)
	}
	return nil
}

func (r *routingRepository) Create(ctx context.Context, rule *domain.RoutingRule) error {
	if err := r.db.WithContext(ctx).Create(rule).Error; err != nil {
		return fmt.Errorf("repository: routing create: %w", err)
	}
	return nil
}

func (r *routingRepository) Update(ctx context.Context, rule *domain.RoutingRule) error {
	if err := r.db.WithContext(ctx).Save(rule).Error; err != nil {
		return fmt.Errorf("repository: routing update: %w", err)
	}
	return nil
}

func (r *routingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&domain.RoutingRule{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("repository: routing delete: %w", err)
	}
	return nil
}

func normalizeDomain(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "www.")
	if i := strings.IndexByte(s, '/'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
