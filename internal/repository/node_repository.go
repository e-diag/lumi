package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type nodeRepository struct {
	db *gorm.DB
}

// NewNodeRepository создаёт реализацию NodeRepository на основе GORM.
func NewNodeRepository(db *gorm.DB) NodeRepository {
	return &nodeRepository{db: db}
}

func (r *nodeRepository) GetAll(ctx context.Context) ([]*domain.Node, error) {
	var nodes []*domain.Node
	if err := r.db.WithContext(ctx).Find(&nodes).Error; err != nil {
		return nil, fmt.Errorf("repository: node get all: %w", err)
	}
	return nodes, nil
}

func (r *nodeRepository) GetByRegion(ctx context.Context, region domain.NodeRegion) ([]*domain.Node, error) {
	var nodes []*domain.Node
	if err := r.db.WithContext(ctx).Where("region = ? AND active = true", region).Find(&nodes).Error; err != nil {
		return nil, fmt.Errorf("repository: node get by region: %w", err)
	}
	return nodes, nil
}

func (r *nodeRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Node, error) {
	var node domain.Node
	err := r.db.WithContext(ctx).First(&node, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNodeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: node get by id: %w", err)
	}
	return &node, nil
}

func (r *nodeRepository) Create(ctx context.Context, node *domain.Node) error {
	if err := r.db.WithContext(ctx).Create(node).Error; err != nil {
		return fmt.Errorf("repository: node create: %w", err)
	}
	return nil
}

func (r *nodeRepository) Update(ctx context.Context, node *domain.Node) error {
	if err := r.db.WithContext(ctx).Save(node).Error; err != nil {
		return fmt.Errorf("repository: node update: %w", err)
	}
	return nil
}
