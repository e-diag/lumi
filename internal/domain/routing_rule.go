package domain

import (
	"time"

	"github.com/google/uuid"
)

// RoutingRuleType — тип routing-правила.
type RoutingRuleType string

const (
	RuleTypeDomain RoutingRuleType = "domain"
	RuleTypeCIDR   RoutingRuleType = "cidr"
)

type RouteAction string

const (
	ActionProxyEU  RouteAction = "proxy_eu"
	ActionProxyUSA RouteAction = "proxy_usa"
	ActionDirect   RouteAction = "direct"
	ActionBlock    RouteAction = "block"
)

// RoutingRule — правило маршрутизации (antifilter / bypass).
type RoutingRule struct {
	ID        uuid.UUID       `gorm:"type:uuid;primaryKey"`
	Type      RoutingRuleType `gorm:"not null"`
	Value     string          `gorm:"not null;size:512;uniqueIndex"` // домен или CIDR
	Source    string          `gorm:"not null;size:64;index"`
	Action    RouteAction     `gorm:"not null;size:32;index"`
	IsManual  bool            `gorm:"not null;default:false;index"`
	Comment   string          `gorm:"size:256"`
	Active    bool            `gorm:"default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type RoutingList struct {
	Version   string
	UpdatedAt time.Time
	ProxyEU   []string
	ProxyUSA  []string
	Direct    []string
	Manual    []string
}

var AIServiceDomains = []string{
	"openai.com", "chat.openai.com", "api.openai.com",
	"anthropic.com", "claude.ai",
	"gemini.google.com", "bard.google.com",
	"midjourney.com", "discord.com",
	"stability.ai", "perplexity.ai",
	"huggingface.co", "replicate.com",
	"together.ai", "groq.com",
}

var DirectDomains = []string{
	"vk.com", "vk.ru", "ok.ru",
	"yandex.ru", "ya.ru", "yandex.com",
	"sber.ru", "sberbank.ru",
	"gosuslugi.ru", "mos.ru",
	"nalog.ru", "pfr.gov.ru",
	"tinkoff.ru", "alfabank.ru", "vtb.ru",
	"mail.ru", "rambler.ru",
	"avito.ru", "wildberries.ru", "ozon.ru",
	"hh.ru", "kinopoisk.ru",
}
