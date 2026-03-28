package usecase

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/freeway-vpn/backend/internal/repository"
	"github.com/google/uuid"
)

type configUseCase struct {
	userRepo repository.UserRepository
	subRepo  repository.SubscriptionRepository
	nodeRepo repository.NodeRepository
}

// NewConfigUseCase создаёт реализацию ConfigUseCase.
func NewConfigUseCase(
	userRepo repository.UserRepository,
	subRepo repository.SubscriptionRepository,
	nodeRepo repository.NodeRepository,
) ConfigUseCase {
	return &configUseCase{
		userRepo: userRepo,
		subRepo:  subRepo,
		nodeRepo: nodeRepo,
	}
}

// GenerateSubscription генерирует base64-закодированный список VLESS-конфигов
// для указанного пользователя на основе его тарифа.
func (uc *configUseCase) GenerateSubscription(ctx context.Context, userUUID uuid.UUID) (string, error) {
	user, err := uc.userRepo.GetByID(ctx, userUUID)
	if err != nil {
		return "", fmt.Errorf("usecase: config generate: %w", err)
	}

	sub, err := uc.subRepo.GetByUserID(ctx, userUUID)
	if err != nil && !errors.Is(err, domain.ErrSubscriptionNotFound) {
		return "", fmt.Errorf("usecase: config generate subscription: %w", err)
	}

	// Определяем тариф: без подписки или истекшая → Free
	tier := domain.TierFree
	if sub != nil && sub.IsActive() {
		tier = sub.Tier
	}

	limits := domain.TierLimitsMap[tier]
	var configs []string

	for _, region := range limits.Regions {
		nodes, err := uc.nodeRepo.GetByRegion(ctx, region)
		if err != nil {
			return "", fmt.Errorf("usecase: config get nodes for region %s: %w", region, err)
		}
		for _, node := range nodes {
			var cfg string
			switch node.Transport {
			case domain.TransportReality:
				cfg = uc.generateVLESSReality(user.ID, node)
			case domain.TransportGRPC:
				cfg = uc.generateVLESSGRPC(user.ID, node)
			case domain.TransportWS:
				cfg = uc.generateVLESSWebSocket(user.ID, node)
			}
			if cfg != "" {
				configs = append(configs, cfg)
			}
		}
	}

	// CDN-конфиг всегда должен быть последним.
	sort.SliceStable(configs, func(i, j int) bool {
		iCDN := strings.Contains(strings.ToLower(configs[i]), "cdn")
		jCDN := strings.Contains(strings.ToLower(configs[j]), "cdn")
		if iCDN == jCDN {
			return i < j
		}
		return !iCDN && jCDN
	})

	if len(configs) == 0 {
		return "", fmt.Errorf("usecase: no active nodes for tier %s", tier)
	}

	raw := strings.Join(configs, "\n")
	return base64.StdEncoding.EncodeToString([]byte(raw)), nil
}

// generateVLESSReality формирует VLESS URI для XTLS Reality транспорта.
// Формат: vless://{uuid}@{host}:{port}?encryption=none&security=reality&...#{name}
func (uc *configUseCase) generateVLESSReality(userUUID uuid.UUID, node *domain.Node) string {
	params := url.Values{}
	params.Set("encryption", "none")
	params.Set("security", "reality")
	params.Set("flow", "xtls-rprx-vision")
	params.Set("type", "tcp")
	params.Set("sni", node.SNI)
	params.Set("pbk", node.PublicKey)
	params.Set("sid", node.ShortID)
	params.Set("fp", "chrome")

	fragment := url.PathEscape(node.Name)
	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		userUUID.String(),
		node.Host,
		node.Port,
		params.Encode(),
		fragment,
	)
}

// generateVLESSGRPC формирует VLESS URI для gRPC CDN-ноды.
// Формат: vless://{uuid}@{host}:{port}?type=grpc&serviceName=vless&security=none#Name_CDN_Backup
func (uc *configUseCase) generateVLESSGRPC(userUUID uuid.UUID, node *domain.Node) string {
	params := url.Values{}
	params.Set("encryption", "none")
	params.Set("type", "grpc")
	if node.GRPCServiceName != "" {
		params.Set("serviceName", node.GRPCServiceName)
	} else {
		params.Set("serviceName", "vless")
	}
	// НЕ tls — TLS уже терминируется на CDN.
	params.Set("security", "none")

	return fmt.Sprintf("vless://%s@%s:%d?%s#%s_CDN_Backup",
		userUUID.String(),
		node.Host,
		node.Port,
		params.Encode(),
		url.QueryEscape(node.Name),
	)
}

// generateVLESSWebSocket формирует VLESS URI для WebSocket+TLS транспорта.
// Оставлено для обратной совместимости со старыми нодами.
func (uc *configUseCase) generateVLESSWebSocket(userUUID uuid.UUID, node *domain.Node) string {
	params := url.Values{}
	params.Set("encryption", "none")
	params.Set("security", "tls")
	params.Set("type", "ws")
	if node.WSHost != "" {
		params.Set("host", node.WSHost)
	} else {
		params.Set("host", node.Host)
	}
	params.Set("sni", node.SNI)
	path := node.WSPath
	if path == "" {
		path = node.Path
	}
	if path != "" {
		params.Set("path", path)
	}
	params.Set("fp", "chrome")

	fragment := url.PathEscape(node.Name)
	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		userUUID.String(),
		node.Host,
		node.Port,
		params.Encode(),
		fragment,
	)
}
