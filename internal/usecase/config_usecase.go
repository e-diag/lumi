package usecase

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/freeway-vpn/backend/internal/infrastructure/vlessconfig"
	"github.com/freeway-vpn/backend/internal/repository"
	"github.com/google/uuid"
)

type configUseCase struct {
	userRepo repository.UserRepository
	subRepo  repository.SubscriptionRepository
	nodeRepo repository.NodeRepository

	cacheTTL time.Duration
	cacheMu  sync.Mutex
	cache    map[string]subCacheEntry
}

type subCacheEntry struct {
	body string
	exp  time.Time
}

// NewConfigUseCase создаёт реализацию ConfigUseCase (кэш подписки ~45 с на пользователя/режим).
func NewConfigUseCase(
	userRepo repository.UserRepository,
	subRepo repository.SubscriptionRepository,
	nodeRepo repository.NodeRepository,
) ConfigUseCase {
	return &configUseCase{
		userRepo: userRepo,
		subRepo:  subRepo,
		nodeRepo: nodeRepo,
		cacheTTL: 45 * time.Second,
		cache:    make(map[string]subCacheEntry),
	}
}

func (uc *configUseCase) getCached(key string) (string, bool) {
	uc.cacheMu.Lock()
	defer uc.cacheMu.Unlock()
	e, ok := uc.cache[key]
	if !ok || time.Now().After(e.exp) {
		if ok {
			delete(uc.cache, key)
		}
		return "", false
	}
	return e.body, true
}

func (uc *configUseCase) setCached(key, body string) {
	uc.cacheMu.Lock()
	defer uc.cacheMu.Unlock()
	if len(uc.cache) > 50000 {
		uc.cache = make(map[string]subCacheEntry)
	}
	uc.cache[key] = subCacheEntry{body: body, exp: time.Now().Add(uc.cacheTTL)}
}

// GenerateSubscription возвращает base64-закодированный список VLESS URI.
// Порядок: приоритет по health_score нод, внутри ноды — Reality → WS → gRPC; CDN в конце, кроме ForceCDN.
func (uc *configUseCase) GenerateSubscription(ctx context.Context, userUUID uuid.UUID) (string, error) {
	user, err := uc.userRepo.GetByID(ctx, userUUID)
	if err != nil {
		return "", fmt.Errorf("usecase: config generate: %w", err)
	}

	sub, err := uc.subRepo.GetByUserID(ctx, userUUID)
	if err != nil && !errors.Is(err, domain.ErrSubscriptionNotFound) {
		return "", fmt.Errorf("usecase: config generate subscription: %w", err)
	}

	tier := domain.TierFree
	if sub != nil && sub.IsActive() {
		tier = sub.Tier
	}

	cacheKey := fmt.Sprintf("%s|%v|%s", userUUID.String(), user.ForceCDN, tier)
	if body, ok := uc.getCached(cacheKey); ok {
		return body, nil
	}

	limits := domain.TierLimitsMap[tier]
	regions := append([]domain.NodeRegion(nil), limits.Regions...)

	emergencyCDN := uc.detectEmergencyPrimaryDown(ctx, regions)
	if emergencyCDN {
		regions = appendUniqueRegion(regions, domain.RegionCDN)
	}

	regions = orderRegionsForForceCDN(regions, user.ForceCDN)

	var lines []string
	for _, region := range regions {
		nodes, err := uc.nodeRepo.GetByRegionWithTopology(ctx, region)
		if err != nil {
			return "", fmt.Errorf("usecase: config topology region %s: %w", region, err)
		}
		sortNodesByHealth(nodes)
		for _, node := range nodes {
			if node == nil || !node.Active {
				continue
			}
			lines = append(lines, uc.buildLinesForNode(userUUID, node)...)
		}
	}

	lines = dedupeLines(lines)
	if user.ForceCDN {
		sortCDNFirst(lines)
	} else {
		sortCDNLast(lines)
	}

	if len(lines) == 0 {
		return "", fmt.Errorf("usecase: no active nodes for tier %s", tier)
	}

	raw := strings.Join(lines, "\n")
	out := base64.StdEncoding.EncodeToString([]byte(raw))
	uc.setCached(cacheKey, out)
	return out, nil
}

func appendUniqueRegion(regions []domain.NodeRegion, r domain.NodeRegion) []domain.NodeRegion {
	for _, x := range regions {
		if x == r {
			return regions
		}
	}
	return append(regions, r)
}

// orderRegionsForForceCDN выносит CDN в начало списка регионов, если пользователь в жёсткой сети.
func orderRegionsForForceCDN(regions []domain.NodeRegion, force bool) []domain.NodeRegion {
	if !force {
		return regions
	}
	var head, tail []domain.NodeRegion
	for _, r := range regions {
		if r == domain.RegionCDN {
			head = append(head, r)
		} else {
			tail = append(tail, r)
		}
	}
	return append(head, tail...)
}

func (uc *configUseCase) detectEmergencyPrimaryDown(ctx context.Context, regions []domain.NodeRegion) bool {
	for _, reg := range regions {
		if reg == domain.RegionCDN {
			continue
		}
		nodes, err := uc.nodeRepo.GetByRegionWithTopology(ctx, reg)
		if err != nil {
			continue
		}
		for _, n := range nodes {
			if n == nil || !n.Active {
				continue
			}
			if hasActiveInboundPlan(n) {
				return false
			}
		}
	}
	return true
}

func hasActiveInboundPlan(n *domain.Node) bool {
	for _, in := range inboundsForNode(n) {
		if in.Active {
			return true
		}
	}
	return false
}

func sortNodesByHealth(nodes []*domain.Node) {
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i] == nil || nodes[j] == nil {
			return false
		}
		if nodes[i].HealthScore != nodes[j].HealthScore {
			return nodes[i].HealthScore > nodes[j].HealthScore
		}
		if nodes[i].LatencyMs != nodes[j].LatencyMs {
			return nodes[i].LatencyMs < nodes[j].LatencyMs
		}
		return nodes[i].Name < nodes[j].Name
	})
}

func inboundsForNode(n *domain.Node) []domain.NodeInbound {
	if len(n.Inbounds) > 0 {
		list := append([]domain.NodeInbound(nil), n.Inbounds...)
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].Priority != list[j].Priority {
				return list[i].Priority < list[j].Priority
			}
			return list[i].Transport < list[j].Transport
		})
		return list
	}
	return legacyInboundSlice(n)
}

func legacyInboundSlice(n *domain.Node) []domain.NodeInbound {
	path := n.WSPath
	if path == "" {
		path = n.Path
	}
	return []domain.NodeInbound{{
		NodeID:          n.ID,
		Transport:       n.Transport,
		ListenPort:      0,
		Path:            path,
		WSHostHeader:    n.WSHost,
		SNI:             n.SNI,
		GRPCServiceName: n.GRPCServiceName,
		Priority:        transportPriorityValue(n.Transport),
		Active:          true,
		UseDomainPool:   true,
	}}
}

func transportPriorityValue(t domain.NodeTransport) int {
	switch t {
	case domain.TransportReality:
		return 10
	case domain.TransportWS:
		return 20
	case domain.TransportGRPC:
		return 30
	default:
		return 50
	}
}

func (uc *configUseCase) buildLinesForNode(userID uuid.UUID, n *domain.Node) []string {
	var out []string
	for _, in := range inboundsForNode(n) {
		if !in.Active {
			continue
		}
		port := in.ListenPort
		if port <= 0 {
			port = n.Port
		}
		connectHost, sniBase := uc.resolveConnectAndSNI(n, &in)
		if connectHost == "" || strings.Contains(connectHost, "${") {
			continue
		}
		sni := chooseSNIFinal(in.SNI, connectHost, sniBase, n.SNI)
		seed := userID.String() + n.ID.String() + string(in.Transport) + connectHost
		fp := vlessconfig.Fingerprint(in.Fingerprint, seed)
		display := fmt.Sprintf("%s · %s", n.Name, in.Transport)

		switch in.Transport {
		case domain.TransportReality:
			if strings.TrimSpace(n.PublicKey) == "" || strings.TrimSpace(n.ShortID) == "" {
				continue
			}
			out = append(out, vlessconfig.BuildReality(userID, connectHost, port, n.PublicKey, n.ShortID, sni, fp, display))
		case domain.TransportWS:
			path := vlessconfig.WebSocketPath(in.Path, seed)
			out = append(out, vlessconfig.BuildWebSocket(userID, connectHost, port, path, in.WSHostHeader, sni, fp, display))
		case domain.TransportGRPC:
			svc := in.GRPCServiceName
			if strings.TrimSpace(svc) == "" {
				svc = n.GRPCServiceName
			}
			if strings.TrimSpace(svc) == "" {
				svc = "vless"
			}
			out = append(out, vlessconfig.BuildGRPC(userID, connectHost, port, svc, n.Name))
		}
	}
	return out
}

func (uc *configUseCase) resolveConnectAndSNI(n *domain.Node, in *domain.NodeInbound) (host string, poolSNI string) {
	if !in.UseDomainPool {
		return strings.TrimSpace(n.Host), strings.TrimSpace(n.Host)
	}
	var candidates []domain.NodeDomain
	for _, d := range n.Domains {
		if d.IsActive && !d.IsBlocked && strings.TrimSpace(d.Domain) != "" {
			candidates = append(candidates, d)
		}
	}
	if len(candidates) == 0 {
		h := strings.TrimSpace(n.Host)
		return h, h
	}
	picked := weightedRandomDomain(candidates)
	return strings.TrimSpace(picked.Domain), strings.TrimSpace(picked.Domain)
}

func weightedRandomDomain(domains []domain.NodeDomain) domain.NodeDomain {
	total := 0
	for _, d := range domains {
		w := d.Weight
		if w < 1 {
			w = 1
		}
		total += w
	}
	r := rand.IntN(total)
	acc := 0
	for _, d := range domains {
		w := d.Weight
		if w < 1 {
			w = 1
		}
		acc += w
		if r < acc {
			return d
		}
	}
	return domains[0]
}

func chooseSNIFinal(inboundSNI, connectHost, fromPool, nodeSNI string) string {
	if strings.TrimSpace(inboundSNI) != "" {
		return strings.TrimSpace(inboundSNI)
	}
	if strings.TrimSpace(fromPool) != "" && strings.Contains(fromPool, ".") {
		return fromPool
	}
	if strings.Contains(connectHost, ".") && !strings.Contains(connectHost, ":") {
		return connectHost
	}
	return strings.TrimSpace(nodeSNI)
}

func isCDNishLine(s string) bool {
	s = strings.ToLower(s)
	return strings.Contains(s, "_cdn_backup") || strings.Contains(s, "type=grpc")
}

func sortCDNLast(lines []string) {
	sort.SliceStable(lines, func(i, j int) bool {
		ci, cj := isCDNishLine(lines[i]), isCDNishLine(lines[j])
		if ci == cj {
			return i < j
		}
		return !ci && cj
	})
}

func sortCDNFirst(lines []string) {
	sort.SliceStable(lines, func(i, j int) bool {
		ci, cj := isCDNishLine(lines[i]), isCDNishLine(lines[j])
		if ci == cj {
			return i < j
		}
		return ci && !cj
	})
}

func dedupeLines(lines []string) []string {
	seen := make(map[string]struct{}, len(lines))
	var out []string
	for _, l := range lines {
		if _, ok := seen[l]; ok {
			continue
		}
		seen[l] = struct{}{}
		out = append(out, l)
	}
	return out
}
