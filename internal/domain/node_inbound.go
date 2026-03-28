package domain

import (
	"time"

	"github.com/google/uuid"
)

// NodeInbound — один способ входа на ноду (Reality / WS+TLS / gRPC).
// Несколько записей на одну Node дают клиенту цепочку fallback без ручного переключения.
type NodeInbound struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	NodeID uuid.UUID `gorm:"type:uuid;not null;index"`

	Transport NodeTransport `gorm:"not null"`
	// ListenPort — порт подключения; 0 означает «взять node.Port».
	ListenPort int `gorm:"not null;default:0"`
	// Path — путь WebSocket или gRPC (service path); для Reality пусто.
	Path string `gorm:"size:256"`
	// WSHostHeader — заголовок Host для WS (пусто → connect host).
	WSHostHeader string `gorm:"size:256"`
	// SNI — фиксированный SNI для Reality/TLS; пусто → берётся из выбранного домена или node.SNI.
	SNI string `gorm:"size:256"`
	// GRPCServiceName — имя gRPC-сервиса (например vless).
	GRPCServiceName string `gorm:"size:128"`
	// Priority — порядок в подписке: меньше = раньше (обычно Reality=10, WS=20, gRPC=30).
	Priority int `gorm:"not null;default:100;index"`
	Active   bool `gorm:"not null;default:true"`
	// UseDomainPool — true: адрес подключения выбирается из node_domains; false: node.Host.
	UseDomainPool bool `gorm:"not null;default:true"`
	// Fingerprint — uTLS fp для Reality (chrome, firefox, …); пусто → выбирается в генераторе.
	Fingerprint string `gorm:"size:32"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
