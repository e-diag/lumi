package xray

import (
	"fmt"
	"net/url"

	"github.com/freeway-vpn/backend/internal/domain"
)

// generateVLESSGRPC генерирует конфиг для CDN-ноды через gRPC.
// gRPC не создает TLS-over-TLS: снаружи это HTTPS до CDN, внутри CDN->origin HTTP/2.
//
// Формат:
// vless://UUID@host:443?encryption=none&type=grpc&serviceName=vless&security=none#Name_CDN_Backup
//
// ВАЖНО: security=none, т.к. TLS терминируется на Яндекс CDN.
func GenerateVLESSGRPC(userUUID string, node *domain.Node) string {
	params := url.Values{}
	params.Set("encryption", "none")
	params.Set("type", "grpc")
	if node.GRPCServiceName == "" {
		params.Set("serviceName", "vless")
	} else {
		params.Set("serviceName", node.GRPCServiceName)
	}
	params.Set("security", "none")

	return fmt.Sprintf("vless://%s@%s:%d?%s#%s_CDN_Backup",
		userUUID,
		node.Host,
		node.Port,
		params.Encode(),
		url.QueryEscape(node.Name),
	)
}

// GenerateVLESSWebSocket оставлен для обратной совместимости.
func GenerateVLESSWebSocket(userUUID string, node *domain.Node) string {
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
	if node.WSPath != "" {
		params.Set("path", node.WSPath)
	}
	params.Set("fp", "chrome")

	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		userUUID,
		node.Host,
		node.Port,
		params.Encode(),
		url.PathEscape(node.Name),
	)
}

