package usecase

import (
	"context"
	"fmt"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/freeway-vpn/backend/internal/repository"
)

type nodeUseCase struct {
	nodeRepo repository.NodeRepository
}

// NewNodeUseCase создаёт реализацию NodeUseCase.
func NewNodeUseCase(nodeRepo repository.NodeRepository) NodeUseCase {
	return &nodeUseCase{nodeRepo: nodeRepo}
}

// GetAllNodes возвращает все ноды.
func (uc *nodeUseCase) GetAllNodes(ctx context.Context) ([]*domain.Node, error) {
	nodes, err := uc.nodeRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("usecase: get all nodes: %w", err)
	}
	return nodes, nil
}

// GetNodesForTier возвращает ноды, доступные для данного тарифа.
func (uc *nodeUseCase) GetNodesForTier(ctx context.Context, tier domain.SubscriptionTier) ([]*domain.Node, error) {
	limits, ok := domain.TierLimitsMap[tier]
	if !ok {
		return nil, fmt.Errorf("usecase: unknown tier: %s", tier)
	}

	var result []*domain.Node
	for _, region := range limits.Regions {
		nodes, err := uc.nodeRepo.GetByRegion(ctx, region)
		if err != nil {
			return nil, fmt.Errorf("usecase: get nodes for tier: %w", err)
		}
		result = append(result, nodes...)
	}
	return result, nil
}

func (uc *nodeUseCase) UpdateNode(ctx context.Context, node *domain.Node) error {
	if err := uc.nodeRepo.Update(ctx, node); err != nil {
		return fmt.Errorf("usecase: update node: %w", err)
	}
	return nil
}
