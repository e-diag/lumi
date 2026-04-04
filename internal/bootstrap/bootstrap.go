// Пакет bootstrap — единая инициализация БД, репозиториев и use case для cmd/*.
package bootstrap

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	"github.com/freeway-vpn/backend/internal/handler/api"
	bothandler "github.com/freeway-vpn/backend/internal/handler/bot"
	webhandler "github.com/freeway-vpn/backend/internal/handler/web"
	"github.com/freeway-vpn/backend/internal/infrastructure/config"
	"github.com/freeway-vpn/backend/internal/infrastructure/database"
	"github.com/freeway-vpn/backend/internal/infrastructure/telegramnotify"
	"github.com/freeway-vpn/backend/internal/infrastructure/xui"
	"github.com/freeway-vpn/backend/internal/infrastructure/yookassa"
	"github.com/freeway-vpn/backend/internal/repository"
	"github.com/freeway-vpn/backend/internal/usecase"
)

// Repositories набор репозиториев на одно подключение к БД.
type Repositories struct {
	User            repository.UserRepository
	Subscription    repository.SubscriptionRepository
	Node            repository.NodeRepository
	Payment         repository.PaymentRepository
	Routing         repository.RoutingRepository
	AccessProbe     repository.AccessProbeRepository
	AntiAbuse       repository.BotAntiAbuseRepository
	Plan            repository.PlanRepository
	ProductSettings repository.ProductSettingsRepository
	VPNServer       repository.VPNServerRepository
}

// NewRepositories создаёт репозитории.
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		User:            repository.NewUserRepository(db),
		Subscription:    repository.NewSubscriptionRepository(db),
		Node:            repository.NewNodeRepository(db),
		Payment:         repository.NewPaymentRepository(db),
		Routing:         repository.NewRoutingRepository(db),
		AccessProbe:     repository.NewAccessProbeRepository(db),
		AntiAbuse:       repository.NewBotAntiAbuseRepository(db),
		Plan:            repository.NewPlanRepository(db),
		ProductSettings: repository.NewProductSettingsRepository(db),
		VPNServer:       repository.NewVPNServerRepository(db),
	}
}

// API зависимости REST-сервера (cmd/api).
type API struct {
	DB     *gorm.DB
	Config *config.Config

	SubHandler     *api.SubHandler
	UserHandler    *api.UserHandler
	PaymentHandler *api.PaymentHandler
	WebhookHandler *api.WebhookHandler
	AuthHandler    *api.AuthHandler
	RoutingHandler *api.RoutingHandler

	PaymentUC usecase.PaymentUseCase
	SubUC     usecase.SubscriptionUseCase
	NodeUC    usecase.NodeUseCase
	NodeRepo  repository.NodeRepository
	RoutingUC usecase.RoutingUseCase
}

// apiBackend — общая инициализация БД и use case для HTTP API и отдельного процесса воркеров.
type apiBackend struct {
	db        *gorm.DB
	cfg       *config.Config
	r         *Repositories
	userUC    usecase.UserUseCase
	subUC     usecase.SubscriptionUseCase
	nodeUC    usecase.NodeUseCase
	paymentUC usecase.PaymentUseCase
	routingUC usecase.RoutingUseCase
	probeUC   usecase.AccessProbeUseCase
	configUC  usecase.ConfigUseCase
}

func newAPIBackend(cfg *config.Config) (*apiBackend, error) {
	if err := cfg.ValidateAPI(); err != nil {
		return nil, err
	}
	db, err := database.Connect(cfg.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: database: %w", err)
	}
	r := NewRepositories(db)

	panelAdapter := newVPNPanelAdapter(cfg)

	userUC := usecase.NewUserUseCase(r.User)
	subUC := usecase.NewSubscriptionUseCase(r.Subscription, r.User, panelAdapter)
	nodeUC := usecase.NewNodeUseCase(r.Node)

	yooClient := yookassa.NewClient(cfg.Yookassa.ShopID, cfg.Yookassa.SecretKey)
	yooGateway := yookassa.NewGatewayAdapter(yooClient)
	payNotify := telegramnotify.NewPaymentSuccessNotifier(cfg.Bot.Token, r.User)
	paymentUC := usecase.NewPaymentUseCase(r.Payment, r.Plan, subUC, yooGateway, cfg.Server.BaseURL, payNotify)

	probeUC := usecase.NewAccessProbeUseCase(r.AccessProbe)
	configUC := usecase.NewConfigUseCase(r.User, r.Subscription, r.Node, cfg.XUI.PublicSubscriptionBaseURL, cfg.XUI.SubscriptionPath)
	routingUC := usecase.NewRoutingUseCase(r.Routing)

	return &apiBackend{
		db:        db,
		cfg:       cfg,
		r:         r,
		userUC:    userUC,
		subUC:     subUC,
		nodeUC:    nodeUC,
		paymentUC: paymentUC,
		routingUC: routingUC,
		probeUC:   probeUC,
		configUC:  configUC,
	}, nil
}

// NewAPI подключается к БД и собирает стек для API.
func NewAPI(cfg *config.Config) (*API, error) {
	b, err := newAPIBackend(cfg)
	if err != nil {
		return nil, err
	}
	return &API{
		DB:             b.db,
		Config:         b.cfg,
		SubHandler:     api.NewSubHandler(b.userUC, b.configUC, "FreeWay VPN", b.probeUC),
		UserHandler:    api.NewUserHandler(b.userUC, b.subUC),
		PaymentHandler: api.NewPaymentHandlerWithSubscription(b.paymentUC, b.subUC),
		WebhookHandler: api.NewWebhookHandler(b.paymentUC),
		AuthHandler:    api.NewAuthHandler(b.userUC, b.cfg.JWT.Secret, b.cfg.Bot.Token),
		RoutingHandler: api.NewRoutingHandler(b.routingUC),
		PaymentUC:      b.paymentUC,
		SubUC:          b.subUC,
		NodeUC:         b.nodeUC,
		NodeRepo:       b.r.Node,
		RoutingUC:      b.routingUC,
	}, nil
}

// Worker — зависимости фоновых воркеров (отдельный процесс cmd/worker, без HTTP).
type Worker struct {
	DB        *gorm.DB
	PaymentUC usecase.PaymentUseCase
	SubUC     usecase.SubscriptionUseCase
	NodeUC    usecase.NodeUseCase
	NodeRepo  repository.NodeRepository
	RoutingUC usecase.RoutingUseCase
}

// NewWorker поднимает тот же стек БД/use case, что и API, но без HTTP-обработчиков.
func NewWorker(cfg *config.Config) (*Worker, error) {
	b, err := newAPIBackend(cfg)
	if err != nil {
		return nil, err
	}
	return &Worker{
		DB:        b.db,
		PaymentUC: b.paymentUC,
		SubUC:     b.subUC,
		NodeUC:    b.nodeUC,
		NodeRepo:  b.r.Node,
		RoutingUC: b.routingUC,
	}, nil
}

// Bot зависимости Telegram-бота (cmd/bot).
type Bot struct {
	DB      *gorm.DB
	Config  *config.Config
	Handler *bothandler.Handler
}

// NewBot подключается к БД и собирает обработчик бота.
func NewBot(cfg *config.Config) (*Bot, error) {
	if err := cfg.ValidateBot(); err != nil {
		return nil, err
	}
	db, err := database.Connect(cfg.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: database: %w", err)
	}
	r := NewRepositories(db)

	userUC := usecase.NewUserUseCase(r.User)
	panelAdapter := newVPNPanelAdapter(cfg)
	subUC := usecase.NewSubscriptionUseCase(r.Subscription, r.User, panelAdapter)

	var yooGateway usecase.PaymentGateway
	paymentsEnabled := strings.TrimSpace(cfg.Yookassa.ShopID) != "" && strings.TrimSpace(cfg.Yookassa.SecretKey) != ""
	if paymentsEnabled {
		yooClient := yookassa.NewClient(cfg.Yookassa.ShopID, cfg.Yookassa.SecretKey)
		yooGateway = yookassa.NewGatewayAdapter(yooClient)
	} else {
		slog.Info("bootstrap: YooKassa not configured; bot runs without in-bot payments (subscriptions and keys still work)")
	}
	paymentUC := usecase.NewPaymentUseCase(r.Payment, r.Plan, subUC, yooGateway, cfg.Server.BaseURL, nil)

	configUC := usecase.NewConfigUseCase(r.User, r.Subscription, r.Node, cfg.XUI.PublicSubscriptionBaseURL, cfg.XUI.SubscriptionPath)
	botUserUC := usecase.NewTelegramBotUserUseCase(r.User, subUC, r.AntiAbuse, cfg.Bot.MaxTrialsPerIP, cfg.Bot.ReferralBonusMaxPerMonth, r.ProductSettings, cfg.Bot.TrialGlobalCapPer24h)
	nodeUC := usecase.NewNodeUseCase(r.Node)
	statsUC := usecase.NewStatsUseCase(r.User, r.Subscription, r.Payment, r.Node, r.VPNServer)
	routingUC := usecase.NewRoutingUseCase(r.Routing)

	pub := bothandler.PublicSettings{
		BaseURL:            cfg.Server.BaseURL,
		BotUsername:        cfg.Bot.Username,
		AppURLIOS:          cfg.Bot.AppURLIOS,
		AppURLAndroid:      cfg.Bot.AppURLAndroid,
		PaymentDefaultDays: cfg.Bot.PaymentDefaultDays,
		SupportURL:         cfg.Bot.SupportURL,
		PaymentsEnabled:    paymentsEnabled,
	}

	h := bothandler.NewHandler(statsUC, userUC, subUC, paymentUC, nodeUC, routingUC, botUserUC, configUC, pub, r.ProductSettings, cfg.Bot.AdminIDs)

	return &Bot{DB: db, Config: cfg, Handler: h}, nil
}

// Web зависимости веб-панели (cmd/web).
type Web struct {
	DB      *gorm.DB
	Config  *config.Config
	Handler *webhandler.WebHandler
}

// NewWeb подключается к БД и собирает веб-обработчик.
// templateDir — каталог с *.html; пустая строка → internal/handler/web/templates относительно cwd.
func NewWeb(cfg *config.Config, templateDir string) (*Web, error) {
	if err := cfg.ValidateWeb(); err != nil {
		return nil, err
	}
	db, err := database.Connect(cfg.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: database: %w", err)
	}
	r := NewRepositories(db)

	userUC := usecase.NewUserUseCase(r.User)
	subUC := usecase.NewSubscriptionUseCase(r.Subscription, r.User, newVPNPanelAdapter(cfg))
	nodeUC := usecase.NewNodeUseCase(r.Node)
	statsUC := usecase.NewStatsUseCase(r.User, r.Subscription, r.Payment, r.Node, r.VPNServer)
	paymentUC := usecase.NewPaymentUseCase(r.Payment, r.Plan, subUC, nil, cfg.Server.BaseURL, nil)
	routingUC := usecase.NewRoutingUseCase(r.Routing)

	if templateDir == "" {
		templateDir = filepath.Join("internal", "handler", "web", "templates")
	}
	wh, err := webhandler.NewWebHandler(statsUC, userUC, subUC, nodeUC, paymentUC, routingUC, r.Plan, r.ProductSettings, r.VPNServer, cfg.Web.AdminToken, templateDir)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: web handler: %w", err)
	}
	return &Web{DB: db, Config: cfg, Handler: wh}, nil
}

// newVPNPanelAdapter возвращает адаптер 3x-ui или nil, если интеграция не настроена.
func newVPNPanelAdapter(cfg *config.Config) usecase.VPNPanelClient {
	base := strings.TrimSpace(cfg.XUI.BaseURL)
	if base == "" {
		slog.Warn("bootstrap: XUI base_url is empty; subscription changes apply to DB only (no 3x-ui API calls)")
		return nil
	}
	if cfg.XUI.InboundID <= 0 {
		slog.Warn("bootstrap: XUI inbound_id is not set; skipping 3x-ui adapter")
		return nil
	}
	a, err := xui.NewAdapter(xui.Config{
		BaseURL:   base,
		Username:  cfg.XUI.Username,
		Password:  cfg.XUI.Password,
		InboundID: cfg.XUI.InboundID,
		LimitIP:   3,
	})
	if err != nil {
		slog.Warn("bootstrap: 3x-ui adapter failed", "error", err)
		return nil
	}
	return a
}
