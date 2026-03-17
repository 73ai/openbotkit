package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	wasrc "github.com/73ai/openbotkit/source/whatsapp"
	"github.com/73ai/openbotkit/store"
)

// whatsAppAuth manages the QR code authentication lifecycle for WhatsApp.
type whatsAppAuth struct {
	mu      sync.Mutex
	qr      string
	linking bool
	syncing bool
	done    bool
}

func (s *Server) handleWhatsAppAuthPage(w http.ResponseWriter, r *http.Request) {
	// Start auth flow if not already running.
	s.startWhatsAppAuth()

	// Adapt the page to poll the server's QR endpoint path.
	page := strings.Replace(wasrc.AuthPage(),
		`fetch("/api/qr")`,
		`fetch("/auth/whatsapp/api/qr")`, 1)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, page)
}

func (s *Server) handleWhatsAppAuthQR(w http.ResponseWriter, r *http.Request) {
	s.waMu.Lock()
	wa := s.waAuth
	s.waMu.Unlock()

	resp := map[string]any{
		"qr":            "",
		"linking":       false,
		"syncing":       false,
		"authenticated": false,
	}

	if wa != nil {
		wa.mu.Lock()
		resp["qr"] = wa.qr
		resp["linking"] = wa.linking
		resp["syncing"] = wa.syncing
		resp["authenticated"] = wa.done
		wa.mu.Unlock()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) startWhatsAppAuth() {
	s.waMu.Lock()
	defer s.waMu.Unlock()

	if s.waAuth != nil && !s.waAuth.done {
		return // already running
	}

	wa := &whatsAppAuth{}
	s.waAuth = wa
	go wa.run(s)
}

func (wa *whatsAppAuth) run(s *Server) {
	ctx := s.ctx
	client, err := wasrc.NewClient(ctx, s.cfg.WhatsAppSessionDBPath())
	if err != nil {
		slog.Error("whatsapp auth: create client", "error", err)
		wa.mu.Lock()
		wa.done = true
		wa.mu.Unlock()
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
		slog.Error("whatsapp auth: connect", "error", err)
	}

	wa.mu.Lock()
	wa.linking = false
	wa.syncing = true
	wa.mu.Unlock()

	var dataDB *store.DB
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: s.cfg.WhatsAppDataDSN()})
	if err == nil {
		if merr := wasrc.Migrate(db); merr == nil {
			dataDB = db
		} else {
			db.Close()
		}
	}
	wasrc.WaitForSync(client, 45, 10, dataDB)
	if dataDB != nil {
		dataDB.Close()
	}

	wa.mu.Lock()
	wa.syncing = false
	wa.done = true
	wa.mu.Unlock()

	slog.Info("whatsapp auth: authentication complete")
}
