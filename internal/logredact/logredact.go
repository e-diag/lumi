// Package logredact — безопасные представления чувствительных значений для структурированных логов.
package logredact

import (
	"fmt"
	"strings"
)

// HTTPPathForLog маскирует сегмент токена в путях вида /sub/{token}, чтобы токен не попадал в access-логи.
func HTTPPathForLog(path string) string {
	if path == "" {
		return path
	}
	if path == "/sub" || path == "/sub/" {
		return "/sub/[redacted]"
	}
	if strings.HasPrefix(path, "/sub/") {
		return "/sub/[redacted]"
	}
	return path
}

// ProviderPaymentIDForLog сокращённый идентификатор для корреляции без полного значения (webhook ЮKassa и т.п.).
func ProviderPaymentIDForLog(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	n := len(id)
	if n <= 8 {
		return fmt.Sprintf("[redacted:len=%d]", n)
	}
	suffix := id[n-4:]
	return fmt.Sprintf("[redacted:len=%d:suffix=%s]", n, suffix)
}
