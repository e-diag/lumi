// Пакет vlessconfig — единая точка сборки VLESS URI для Xray-совместимых клиентов.
package vlessconfig

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

// Список отпечатков uTLS, из которого псевдослучайно выбирается значение (снижает статический след).
var realityFingerprints = []string{"chrome", "firefox", "safari", "edge", "randomized"}

// Типовые пути WS, если в inbound не задан свой (выглядят как обычный трафик API/CDN).
var defaultWSPaths = []string{"/api/v1/data", "/cdn/assets", "/images", "/static/stream"}

// Fingerprint выбирает fp для Reality: override непустой — как есть, иначе детерминированно от seed.
func Fingerprint(override, seed string) string {
	if strings.TrimSpace(override) != "" {
		return strings.TrimSpace(override)
	}
	h := sha256.Sum256([]byte(seed))
	idx := binary.BigEndian.Uint64(h[:8]) % uint64(len(realityFingerprints))
	return realityFingerprints[idx]
}

// WebSocketPath возвращает path для WS: из конфигурации или псевдослучайный «нормальный» путь.
func WebSocketPath(configured, seed string) string {
	if strings.TrimSpace(configured) != "" {
		return configured
	}
	h := sha256.Sum256([]byte("ws:" + seed))
	idx := binary.BigEndian.Uint64(h[8:16]) % uint64(len(defaultWSPaths))
	return defaultWSPaths[idx]
}

// BuildReality собирает vless:// для VLESS + REALITY + tcp + vision.
func BuildReality(userUUID uuid.UUID, connectHost string, port int, publicKey, shortID, sni, fp, displayName string) string {
	params := url.Values{}
	params.Set("encryption", "none")
	params.Set("security", "reality")
	params.Set("flow", "xtls-rprx-vision")
	params.Set("type", "tcp")
	params.Set("sni", sni)
	params.Set("pbk", publicKey)
	params.Set("sid", shortID)
	params.Set("fp", fp)

	fragment := url.PathEscape(displayName)
	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		userUUID.String(),
		connectHost,
		port,
		params.Encode(),
		fragment,
	)
}

// BuildWebSocket — VLESS + WS + TLS.
func BuildWebSocket(userUUID uuid.UUID, connectHost string, port int, path, hostHeader, sni, fp, displayName string) string {
	params := url.Values{}
	params.Set("encryption", "none")
	params.Set("security", "tls")
	params.Set("type", "ws")
	if strings.TrimSpace(hostHeader) != "" {
		params.Set("host", hostHeader)
	} else {
		params.Set("host", connectHost)
	}
	params.Set("sni", sni)
	if path != "" {
		params.Set("path", path)
	}
	params.Set("fp", fp)

	fragment := url.PathEscape(displayName)
	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		userUUID.String(),
		connectHost,
		port,
		params.Encode(),
		fragment,
	)
}

// BuildGRPC — VLESS + gRPC (часто за CDN, TLS на edge). Формат имени совместим со старой подпиской.
func BuildGRPC(userUUID uuid.UUID, connectHost string, port int, serviceName, displayName string) string {
	params := url.Values{}
	params.Set("encryption", "none")
	params.Set("type", "grpc")
	if serviceName != "" {
		params.Set("serviceName", serviceName)
	} else {
		params.Set("serviceName", "vless")
	}
	params.Set("security", "none")

	frag := url.QueryEscape(displayName) + "_CDN_Backup"
	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		userUUID.String(),
		connectHost,
		port,
		params.Encode(),
		frag,
	)
}
