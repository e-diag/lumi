package database

import (
	"fmt"
	"strings"
	"time"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// backfillNodeTopology создаёт node_domains и node_inbounds для существующих нод (идемпотентно).
func backfillNodeTopology(db *gorm.DB) error {
	var inboundCount int64
	if err := db.Model(&domain.NodeInbound{}).Count(&inboundCount).Error; err != nil {
		return fmt.Errorf("backfill: count inbounds: %w", err)
	}
	if inboundCount > 0 {
		return nil
	}

	var nodes []domain.Node
	if err := db.Find(&nodes).Error; err != nil {
		return fmt.Errorf("backfill: list nodes: %w", err)
	}

	for i := range nodes {
		n := &nodes[i]
		if err := ensureNodeDomains(db, n); err != nil {
			return err
		}
		if err := ensureNodeInbounds(db, n); err != nil {
			return err
		}
	}
	return nil
}

func ensureNodeDomains(db *gorm.DB, n *domain.Node) error {
	var dc int64
	if err := db.Model(&domain.NodeDomain{}).Where("node_id = ?", n.ID).Count(&dc).Error; err != nil {
		return err
	}
	if dc > 0 {
		return nil
	}
	host := strings.TrimSpace(n.Host)
	if host == "" || strings.Contains(host, "${") {
		return nil
	}
	now := time.Now()
	row := &domain.NodeDomain{
		ID:        uuid.New(),
		NodeID:    n.ID,
		Domain:    host,
		IsActive:  true,
		IsBlocked: false,
		Weight:    10,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return db.Create(row).Error
}

func ensureNodeInbounds(db *gorm.DB, n *domain.Node) error {
	now := time.Now()
	priority := transportPriority(n.Transport)
	primary := domain.NodeInbound{
		ID:              uuid.New(),
		NodeID:          n.ID,
		Transport:       n.Transport,
		ListenPort:      0,
		Path:            firstNonEmpty(n.WSPath, n.Path),
		WSHostHeader:    n.WSHost,
		SNI:             n.SNI,
		GRPCServiceName: n.GRPCServiceName,
		Priority:        priority,
		Active:          true,
		UseDomainPool:   true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(&primary).Error; err != nil {
		return fmt.Errorf("backfill inbound primary: %w", err)
	}

	// Дополнительные транспорты на той же ноде: клиент перебирает URI; на сервере должны быть подняты соответствующие inbounds.
	if n.Region != domain.RegionCDN && n.Transport == domain.TransportReality {
		ws := domain.NodeInbound{
			ID:            uuid.New(),
			NodeID:        n.ID,
			Transport:     domain.TransportWS,
			ListenPort:    0,
			Path:          "",
			WSHostHeader:  "",
			SNI:           "",
			Priority:      20,
			Active:        true,
			UseDomainPool: true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := db.Create(&ws).Error; err != nil {
			return fmt.Errorf("backfill inbound ws: %w", err)
		}
		grpc := domain.NodeInbound{
			ID:              uuid.New(),
			NodeID:          n.ID,
			Transport:       domain.TransportGRPC,
			ListenPort:      0,
			GRPCServiceName: firstNonEmpty(n.GRPCServiceName, "vless"),
			Priority:        30,
			Active:          true,
			UseDomainPool:   true,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := db.Create(&grpc).Error; err != nil {
			return fmt.Errorf("backfill inbound grpc: %w", err)
		}
	}
	return nil
}

func transportPriority(t domain.NodeTransport) int {
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

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
