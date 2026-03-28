// Пакет database инициализирует соединение с PostgreSQL через GORM,
// выполняет автомиграцию схемы и засевает начальные данные нод.
package database

import (
	"fmt"
	"log/slog"

	"github.com/freeway-vpn/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect устанавливает соединение с PostgreSQL и возвращает экземпляр GORM.
func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("database: connect: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("database: migrate: %w", err)
	}

	if err := seed(db); err != nil {
		return nil, fmt.Errorf("database: seed: %w", err)
	}

	if err := backfillNodeTopology(db); err != nil {
		return nil, fmt.Errorf("database: topology backfill: %w", err)
	}

	slog.Info("database connected and migrated")
	return db, nil
}

// migrate выполняет автомиграцию всех domain-моделей.
func migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&domain.User{},
		&domain.Subscription{},
		&domain.Node{},
		&domain.NodeInbound{},
		&domain.NodeDomain{},
		&domain.Payment{},
		&domain.PaymentActivation{},
		&domain.RoutingRule{},
		&domain.MigrationRecord{},
		&domain.BotTrialSignup{},
		&domain.ReferralBonusLog{},
		&domain.UserAccessProbe{},
	); err != nil {
		return err
	}
	return ensurePaymentIndexes(db)
}

// ensurePaymentIndexes добавляет составные индексы для воркеров и отчётов (идемпотентно).
func ensurePaymentIndexes(db *gorm.DB) error {
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_payments_status_created_at
		ON payments (status, created_at);
	`).Error; err != nil {
		return fmt.Errorf("database: index payments status_created_at: %w", err)
	}
	return nil
}

// seed создаёт начальные ноды, если их ещё нет в БД.
func seed(db *gorm.DB) error {
	var count int64
	db.Model(&domain.Node{}).Count(&count)
	if count > 0 {
		return nil
	}

	nodes := []domain.Node{
		{
			ID:        uuid.New(),
			Name:      "EU-NL (Hetzner)",
			Host:      "${NODE_EU_HOST}",
			Port:      443,
			Region:    domain.RegionEU,
			Transport: domain.TransportReality,
			PublicKey: "${NODE_EU_PUBLIC_KEY}",
			ShortID:   "${NODE_EU_SHORT_ID}",
			SNI:       "www.google.com",
			Active:    true,
		},
		{
			ID:        uuid.New(),
			Name:      "USA (Vultr)",
			Host:      "${NODE_USA_HOST}",
			Port:      443,
			Region:    domain.RegionUSA,
			Transport: domain.TransportReality,
			PublicKey: "${NODE_USA_PUBLIC_KEY}",
			ShortID:   "${NODE_USA_SHORT_ID}",
			SNI:       "www.google.com",
			Active:    true,
		},
		{
			ID:        uuid.New(),
			Name:      "CDN (Yandex Cloud)",
			Host:      "${NODE_CDN_HOST}",
			Port:      443,
			Region:    domain.RegionCDN,
			Transport: domain.TransportGRPC,
			SNI:       "${NODE_CDN_SNI}",
			GRPCServiceName: "vless",
			Active:    true,
		},
	}

	if err := db.Create(&nodes).Error; err != nil {
		return fmt.Errorf("seed nodes: %w", err)
	}

	slog.Info("nodes seeded", "count", len(nodes))

	// Seed базовых direct-доменов (upsert по value).
	additionalDirectDomains := []string{
		"vk.com", "vk.ru", "vkontakte.ru",
		"ok.ru", "odnoklassniki.ru",
		"yandex.ru", "yandex.com", "ya.ru",
		"mail.ru", "bk.ru", "list.ru", "inbox.ru",
		"rambler.ru",
		"gosuslugi.ru", "mos.ru", "nalog.ru",
		"pfr.gov.ru", "fss.ru", "mvd.ru",
		"fsb.ru", "rkn.gov.ru",
		"sber.ru", "sberbank.ru", "sberbankl.ru",
		"vtb.ru", "alfabank.ru", "tinkoff.ru",
		"raiffeisen.ru", "gazprombank.ru",
		"ozon.ru", "wildberries.ru", "wb.ru",
		"avito.ru", "cian.ru", "hh.ru",
		"rutube.ru", "rbc.ru", "kommersant.ru",
		"ria.ru", "tass.ru", "interfax.ru",
		"icq.com", "agent.mail.ru",
		"akamai.com", "akamaiedge.net",
		"yandex-team.ru", "yastatic.net", "yandex.net",
		"maps.yandex.ru", "music.yandex.ru",
		"disk.yandex.ru", "cloud.yandex.ru",
	}
	for _, d := range additionalDirectDomains {
		rule := &domain.RoutingRule{
			ID:       uuid.New(),
			Type:     domain.RuleTypeDomain,
			Value:    d,
			Source:   "seed_direct",
			Action:   domain.ActionDirect,
			IsManual: false,
			Active:   true,
		}
		if err := db.Where("value = ?", d).Assign(rule).FirstOrCreate(rule).Error; err != nil {
			return fmt.Errorf("seed direct routing domains: %w", err)
		}
	}

	return nil
}
