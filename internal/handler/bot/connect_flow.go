package bot

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/go-telegram/bot/models"
)

// connectPlatformKeyboard — выбор платформы перед подключением.
func connectPlatformKeyboard() models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "📱 iPhone", CallbackData: cbConnIOS}},
			{{Text: "📱 Android", CallbackData: cbConnAnd}},
			{{Text: "← Меню", CallbackData: cbUserMenu}},
		},
	}
}

// connectDetailKeyboard — скачать приложение, открыть импорт, подсказка со ссылкой.
func connectDetailKeyboard(subURL, storeIOS, storeAndroid, platform string) models.InlineKeyboardMarkup {
	var rows [][]models.InlineKeyboardButton
	platform = strings.ToLower(strings.TrimSpace(platform))

	if platform == "ios" {
		if s := strings.TrimSpace(storeIOS); s != "" {
			rows = append(rows, []models.InlineKeyboardButton{
				{Text: "📲 Скачать V2Box", URL: s},
			})
		}
		if dl := v2boxDeepLink(subURL); dl != "" {
			rows = append(rows, []models.InlineKeyboardButton{
				{Text: "🚀 Подключиться", URL: dl},
			})
		}
	}
	if platform == "android" {
		if s := strings.TrimSpace(storeAndroid); s != "" {
			rows = append(rows, []models.InlineKeyboardButton{
				{Text: "📲 Скачать v2rayNG", URL: s},
			})
		}
		if dl := v2rayNGDeepLink(subURL); dl != "" {
			rows = append(rows, []models.InlineKeyboardButton{
				{Text: "🚀 Подключиться", URL: dl},
			})
		}
	}

	rows = append(rows, []models.InlineKeyboardButton{
		{Text: "📋 Показать ссылку", CallbackData: cbConnCopy},
	})
	rows = append(rows, []models.InlineKeyboardButton{
		{Text: "← Другой телефон", CallbackData: cbUserConn},
		{Text: "🏠 Меню", CallbackData: cbUserMenu},
	})
	return models.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func v2rayNGDeepLink(subURL string) string {
	if strings.TrimSpace(subURL) == "" {
		return ""
	}
	enc := base64.StdEncoding.EncodeToString([]byte(subURL))
	return "v2rayng://install-config?url=" + enc
}

func v2boxDeepLink(subURL string) string {
	if strings.TrimSpace(subURL) == "" {
		return ""
	}
	enc := base64.StdEncoding.EncodeToString([]byte(subURL))
	// Схема совместима с рядом клиентов на iOS; при необходимости замените в одном месте.
	return "v2box://install-config?url=" + enc
}

func connectInstructionText(platform, subURL string) string {
	platform = strings.ToLower(strings.TrimSpace(platform))
	name := "приложение"
	if platform == "ios" {
		name = "V2Box"
	}
	if platform == "android" {
		name = "v2rayNG"
	}
	return fmt.Sprintf(
		"🔌 Подключение\n\n"+
			"1) При необходимости установите %s (кнопка «Скачать»).\n"+
			"2) Нажмите «Подключиться» — ссылка откроется в приложении.\n\n"+
			"Если кнопка не сработала: нажмите «Показать ссылку», затем удерживайте её и вставьте в %s (импорт по ссылке).\n\n"+
			"Ваша ссылка:\n%s",
		name, name, subURL,
	)
}
