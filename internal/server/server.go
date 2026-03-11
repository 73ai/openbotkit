package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	tgchannel "github.com/priyanshujain/openbotkit/channel/telegram"
	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/memory"
	"github.com/priyanshujain/openbotkit/provider"
	ansrc "github.com/priyanshujain/openbotkit/source/applenotes"
	gmailsrc "github.com/priyanshujain/openbotkit/source/gmail"
	historysrc "github.com/priyanshujain/openbotkit/source/history"
	imsrc "github.com/priyanshujain/openbotkit/source/imessage"
	usagesrc "github.com/priyanshujain/openbotkit/source/usage"
	wasrc "github.com/priyanshujain/openbotkit/source/whatsapp"
	"github.com/priyanshujain/openbotkit/store"

	// Register provider factories.
	_ "github.com/priyanshujain/openbotkit/provider/anthropic"
	_ "github.com/priyanshujain/openbotkit/provider/gemini"
	_ "github.com/priyanshujain/openbotkit/provider/openai"
)

type Server struct {
	cfg  *config.Config
	addr string
	ctx  context.Context

	waMu   sync.Mutex
	waAuth *whatsAppAuth
}

func New(cfg *config.Config, addr string) *Server {
	return &Server{cfg: cfg, addr: addr}
}

func (s *Server) Run(ctx context.Context) error {
	s.ctx = ctx

	u, p := s.authCredentials()
	if u == "" || p == "" {
		return fmt.Errorf("server requires authentication credentials; set OBK_AUTH_USERNAME and OBK_AUTH_PASSWORD env vars or configure auth in config.yaml")
	}

	s.migrateDBs()

	mux := http.NewServeMux()
	s.routes(mux)

	srv := &http.Server{
		Addr:              s.addr,
		Handler:           limitBody(mux),
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", s.addr)
		errCh <- srv.ListenAndServe()
	}()

	// Start Telegram bot if configured.
	if err := s.startTelegram(ctx); err != nil {
		slog.Warn("telegram not started", "error", err)
	}

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		slog.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func (s *Server) startTelegram(ctx context.Context) error {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" && s.cfg.Channels != nil && s.cfg.Channels.Telegram != nil {
		token = s.cfg.Channels.Telegram.BotToken
	}
	if token == "" {
		return fmt.Errorf("no telegram bot token configured")
	}

	var ownerID int64
	if idStr := os.Getenv("TELEGRAM_OWNER_ID"); idStr != "" {
		var err error
		ownerID, err = strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return fmt.Errorf("parse TELEGRAM_OWNER_ID: %w", err)
		}
	} else if s.cfg.Channels != nil && s.cfg.Channels.Telegram != nil {
		ownerID = s.cfg.Channels.Telegram.OwnerID
	}
	if ownerID == 0 {
		return fmt.Errorf("no telegram owner ID configured")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return fmt.Errorf("create telegram bot: %w", err)
	}

	// Resolve the default model's provider.
	registry, err := provider.NewRegistry(s.cfg.Models)
	if err != nil {
		return fmt.Errorf("create provider registry: %w", err)
	}

	providerName, modelName, err := provider.ParseModelSpec(s.cfg.Models.Default)
	if err != nil {
		return fmt.Errorf("parse model spec: %w", err)
	}
	p, ok := registry.Get(providerName)
	if !ok {
		return fmt.Errorf("provider %q not found", providerName)
	}

	ch := tgchannel.NewChannel(bot, ownerID)
	poller := tgchannel.NewPoller(bot, ownerID, ch)
	sm := tgchannel.NewSessionManager(s.cfg, ch, p, providerName, modelName)

	go poller.Run(ctx)
	go sm.Run(ctx)

	slog.Info("telegram bot started", "owner_id", ownerID)
	return nil
}

// migrateDBs runs database migrations once at startup for all configured sources.
func (s *Server) migrateDBs() {
	migrations := []struct {
		name   string
		driver string
		dsn    string
		fn     func(*store.DB) error
	}{
		{"memory", s.cfg.UserMemory.Storage.Driver, s.cfg.UserMemoryDataDSN(), memory.Migrate},
		{"history", s.cfg.History.Storage.Driver, s.cfg.HistoryDataDSN(), historysrc.Migrate},
		{"applenotes", s.cfg.AppleNotes.Storage.Driver, s.cfg.AppleNotesDataDSN(), ansrc.Migrate},
		{"imessage", s.cfg.IMessage.Storage.Driver, s.cfg.IMessageDataDSN(), imsrc.Migrate},
		{"whatsapp", s.cfg.WhatsApp.Storage.Driver, s.cfg.WhatsAppDataDSN(), wasrc.Migrate},
		{"gmail", s.cfg.Gmail.Storage.Driver, s.cfg.GmailDataDSN(), gmailsrc.Migrate},
		{"usage", s.cfg.Usage.Storage.Driver, s.cfg.UsageDataDSN(), usagesrc.Migrate},
	}
	for _, m := range migrations {
		db, err := store.Open(store.Config{Driver: m.driver, DSN: m.dsn})
		if err != nil {
			slog.Warn("migrate: open failed", "source", m.name, "error", err)
			continue
		}
		if err := m.fn(db); err != nil {
			slog.Warn("migrate: failed", "source", m.name, "error", err)
		}
		db.Close()
	}
}
