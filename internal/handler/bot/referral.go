package bot

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ParseReferralUserID извлекает UUID пригласившего из текста /start (payload ref_<uuid>).
func ParseReferralUserID(startMessageText string) *uuid.UUID {
	text := strings.TrimSpace(startMessageText)
	parts := strings.Fields(text)
	if len(parts) < 2 {
		return nil
	}
	raw := strings.TrimPrefix(parts[1], "ref_")
	if raw == parts[1] {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &id
}

// BuildReferralLink формирует deep link для приглашения (username без @).
func BuildReferralLink(botUsername string, inviterUserID uuid.UUID) string {
	u := strings.TrimPrefix(strings.TrimSpace(botUsername), "@")
	if u == "" {
		return ""
	}
	return fmt.Sprintf("https://t.me/%s?start=ref_%s", u, inviterUserID.String())
}
