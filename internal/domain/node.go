package domain

import "github.com/google/uuid"

// NodeRegion — географический регион ноды.
type NodeRegion string

const (
	RegionEU  NodeRegion = "eu"
	RegionUSA NodeRegion = "usa"
	RegionCDN NodeRegion = "cdn"
)

// NodeTransport — протокол транспорта ноды.
type NodeTransport string

const (
	TransportReality NodeTransport = "reality" // основной, VLESS+XTLS-Reality
	TransportGRPC    NodeTransport = "grpc"    // CDN-fallback, лучше проходит DPI
	TransportWS      NodeTransport = "ws"      // устаревший CDN, оставить для совместимости
	// Совместимость со старым именем из предыдущих фаз.
	TransportWebSocket = TransportWS
)

// Node — VPN-нода (сервер).
type Node struct {
	ID        uuid.UUID     `gorm:"type:uuid;primaryKey"`
	Name      string        `gorm:"not null"`
	Host      string        `gorm:"not null"`       // IP или домен
	Port      int           `gorm:"not null"`
	Region    NodeRegion    `gorm:"not null;index"`
	Transport NodeTransport `gorm:"not null"`
	PublicKey string        `gorm:"size:256"`       // для Reality
	ShortID   string        `gorm:"size:64"`        // для Reality
	SNI       string        `gorm:"size:256"`       // ServerName для Reality/TLS
	// gRPC параметры (для CDN-ноды)
	GRPCServiceName string `gorm:"size:128"`
	// WebSocket параметры (устаревшие, оставить для совместимости)
	WSPath string `gorm:"size:256"`
	WSHost string `gorm:"size:256"`
	// Legacy alias для старого поля (используется в старом коде/тестах).
	Path string `gorm:"-"`
	Active    bool          `gorm:"default:true"`
	LatencyMs int           `gorm:"not null;default:0"`
	FailCount int           `gorm:"not null;default:0"` // подряд неудачных проверок
}
