package bot

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/go-telegram/bot/models"
)

// connectPlatformKeyboard — выбор клиента: Happ (iOS) или v2RayTun (Android).
func connectPlatformKeyboard() models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "🍎 Happ (iOS)", CallbackData: cbConnHapp}},
			{{Text: "🤖 v2RayTun (Android)", CallbackData: cbConnV2}},
			{{Text: "← Меню", CallbackData: cbUserMenu}},
		},
	}
}

// connectDetailKeyboard — скачать приложение, импорт подписки, показать ссылку.
func connectDetailKeyboard(subURL, storeHapp, storeV2, platform string) models.InlineKeyboardMarkup {
	var rows [][]models.InlineKeyboardButton
	platform = strings.ToLower(strings.TrimSpace(platform))

	if platform == "happ" {
		if s := strings.TrimSpace(storeHapp); s != "" {
			rows = append(rows, []models.InlineKeyboardButton{
				{Text: "📲 Скачать Happ", URL: s},
			})
		}
		if dl := happImportDeepLink(subURL); dl != "" {
			rows = append(rows, []models.InlineKeyboardButton{
				{Text: "🚀 Импорт подписки", URL: dl},
			})
		}
	}
	if platform == "v2raytun" {
		if s := strings.TrimSpace(storeV2); s != "" {
			rows = append(rows, []models.InlineKeyboardButton{
				{Text: "📲 Скачать v2RayTun", URL: s},
			})
		}
		if dl := v2RayTunDeepLink(subURL); dl != "" {
			rows = append(rows, []models.InlineKeyboardButton{
				{Text: "🚀 Импорт подписки", URL: dl},
			})
		}
	}

	rows = append(rows, []models.InlineKeyboardButton{
		{Text: "📋 Показать ссылку", CallbackData: cbConnCopy},
	})
	rows = append(rows, []models.InlineKeyboardButton{
		{Text: "← Другое приложение", CallbackData: cbUserConn},
		{Text: "🏠 Меню", CallbackData: cbUserMenu},
	})
	return models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// v2RayTunDeepLink — схема совместима с рядом форков v2ray на Android (как v2rayNG).
func v2RayTunDeepLink(subURL string) string {
	if strings.TrimSpace(subURL) == "" {
		return ""
	}
	enc := base64.StdEncoding.EncodeToString([]byte(subURL))
	return "v2raytun://install-config?url=" + enc
}

// happImportDeepLink — открываем HTTPS subscription URL в браузере; дальше импорт в Happ из буфера или «Открыть в…».
func happImportDeepLink(subURL string) string {
	u := strings.TrimSpace(subURL)
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}
	return ""
}

func connectInstructionText(platform, subURL string) string {
	platform = strings.ToLower(strings.TrimSpace(platform))
	name := "клиент"
	hint := "вставьте ссылку вручную в разделе подписки."
	switch platform {
	case "happ":
		name = "Happ"
		hint = "в Happ: Подписка → «+» → «Из URL» / вставьте ссылку."
	case "v2raytun":
		name = "v2RayTun"
		hint = "в v2RayTun: группы → «+» → «Подписка» → вставьте URL."
	}
	return fmt.Sprintf(
		"🔌 Подключение · %s\n\n"+
			"1) При необходимости установите приложение (кнопка «Скачать»).\n"+
			"2) Нажмите «Импорт подписки» или откройте ссылку в браузере.\n"+
			"3) Если не открылось — %s\n\n"+
			"Ссылка подписки:\n%s",
		name,
		hint,
		subURL,
	)
}
