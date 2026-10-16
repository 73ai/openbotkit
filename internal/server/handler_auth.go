package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/73ai/openbotkit/config"
	wasrc "github.com/73ai/openbotkit/source/whatsapp"
	"github.com/73ai/openbotkit/store"
)

// whatsAppAuth manages the QR code authentication lifecycle for one WhatsApp account.
type whatsAppAuth struct {
	account string
	entry   *config.WhatsAppAccount
	mu      sync.Mutex
	qr      string
	linking bool
	syncing bool
	done    bool
	err     string
}

func whatsAppAccountParam(r *http.Request) string {
	if a := r.URL.Query().Get("account"); a != "" {
		return a
	}
	return wasrc.DefaultAccount
}

func (s *Server) handleWhatsAppAuthPage(w http.ResponseWriter, r *http.Request) {
	account := whatsAppAccountParam(r)
	s.cfgMu.Lock()
	entry, err := s.cfg.ResolveWhatsAppAccount(account, r.URL.Query().Get("role"), r.URL.Query().Get("owner_jid"))
	s.cfgMu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.startWhatsAppAuth(account, entry)

	page := strings.Replace(wasrc.AuthPage(),
		`fetch("/api/qr")`,
		`fetch("/auth/whatsapp/api/qr?account=`+url.QueryEscape(account)+`")`, 1)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, page)
}

func (s *Server) handleWhatsAppAuthQR(w http.ResponseWriter, r *http.Request) {
	s.waMu.Lock()
	wa := s.waAuth[whatsAppAccountParam(r)]
	s.waMu.Unlock()

	resp := map[string]any{
		"qr":            "",
		"linking":       false,
		"syncing":       false,
		"authenticated": false,
		"error":         "",
	}

	if wa != nil {
		wa.mu.Lock()
		resp["qr"] = wa.qr
		resp["linking"] = wa.linking
		resp["syncing"] = wa.syncing
		resp["authenticated"] = wa.done && wa.err == ""
		resp["error"] = wa.err
		wa.mu.Unlock()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) startWhatsAppAuth(account string, entry *config.WhatsAppAccount) {
	s.waMu.Lock()
	defer s.waMu.Unlock()

	if wa := s.waAuth[account]; wa != nil && !wa.isDone() {
		return
	}
	if s.waAuth == nil {
		s.waAuth = map[string]*whatsAppAuth{}
	}
	wa := &whatsAppAuth{account: account, entry: entry}
	s.waAuth[account] = wa
	go wa.run(s)
}

func (wa *whatsAppAuth) isDone() bool {
	wa.mu.Lock()
	defer wa.mu.Unlock()
	return wa.done
}

func (wa *whatsAppAuth) finish(err error) {
	wa.mu.Lock()
	defer wa.mu.Unlock()
	wa.linking, wa.syncing, wa.done = false, false, true
	if err != nil {
		wa.err = err.Error()
		slog.Error("whatsapp auth failed", "account", wa.account, "error", err)
		return
	}
	slog.Info("whatsapp auth: authentication complete", "account", wa.account)
}

func (wa *whatsAppAuth) run(s *Server) {
	ctx := s.ctx
	sessionPath := s.cfg.WhatsAppAccountSessionDBPath(wa.account)
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0700); err != nil {
		wa.finish(fmt.Errorf("create account dir: %w", err))
		return
	}
	client, err := wasrc.NewClient(ctx, sessionPath)
	if err != nil {
		wa.finish(fmt.Errorf("create client: %w", err))
		return
	}
	defer client.Disconnect()

	qrChan := make(chan string, 5)
	go func() {
		for qr := range qrChan {
			wa.mu.Lock()
			wa.qr = qr
			wa.mu.Unlock()
		}
		wa.mu.Lock()
		wa.linking = true
		wa.mu.Unlock()
	}()

	if err := client.ConnectWithQR(ctx, qrChan); err != nil {
		wa.finish(err)
		return
	}

	wa.mu.Lock()
	wa.linking = false
	wa.syncing = true
	wa.mu.Unlock()

	db, err := store.Open(store.Config{Driver: s.cfg.WhatsApp.Storage.Driver, DSN: s.cfg.WhatsAppDataDSN()})
	if err != nil {
		wa.finish(fmt.Errorf("open data db: %w", err))
		return
	}
	defer db.Close()
	if err := wasrc.Migrate(db); err != nil {
		wa.finish(fmt.Errorf("migrate data db: %w", err))
		return
	}
	wasrc.WaitForSync(client, wa.account, 45, 10, db)
	client.Disconnect()

	if err := s.registerWhatsAppAccount(wa.account, wa.entry); err != nil {
		wa.finish(err)
		return
	}
	s.startRegisteredWhatsAppChannel(ctx, wa.account)
	wa.finish(nil)
}

func (s *Server) startRegisteredWhatsAppChannel(ctx context.Context, label string) {
	s.cfgMu.Lock()
	entries := s.cfg.WhatsAppAccountList()
	s.cfgMu.Unlock()
	for _, e := range entries {
		if e.Label == label {
			s.startWhatsAppChannel(ctx, e)
		}
	}
}

func (s *Server) registerWhatsAppAccount(account string, entry *config.WhatsAppAccount) error {
	if err := config.LinkWhatsAppAccount(account); err != nil {
		return fmt.Errorf("link account: %w", err)
	}
	if entry == nil {
		return nil
	}
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	s.cfg.SetWhatsAppAccount(account, *entry)
	if err := s.cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}
