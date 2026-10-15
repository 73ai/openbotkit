package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/73ai/openbotkit/config"
)

func qrStatus(t *testing.T, s *Server, query string) map[string]any {
	t.Helper()
	rec := httptest.NewRecorder()
	s.handleWhatsAppAuthQR(rec, httptest.NewRequest("GET", "/auth/whatsapp/api/qr"+query, nil))
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp
}

func TestWhatsAppAuthPageRejectsInvalidAccountWithoutStartingLogin(t *testing.T) {
	s := &Server{cfg: config.Default()}
	for _, query := range []string{"?account=assistant&role=channel", "?account=../x", "?account=work&role=reader"} {
		rec := httptest.NewRecorder()
		s.handleWhatsAppAuthPage(rec, httptest.NewRequest("GET", "/auth/whatsapp"+query, nil))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", query, rec.Code)
		}
	}
	if len(s.waAuth) != 0 {
		t.Fatalf("login started for invalid input: %v", s.waAuth)
	}
}

func TestWhatsAppAuthQRReportsFailurePerAccount(t *testing.T) {
	work := &whatsAppAuth{account: "work"}
	work.finish(errors.New("whatsapp login failed (err-client-outdated)"))
	def := &whatsAppAuth{account: "default", qr: "2@abc"}
	s := &Server{cfg: config.Default(), waAuth: map[string]*whatsAppAuth{"work": work, "default": def}}

	got := qrStatus(t, s, "?account=work")
	if got["authenticated"] != false || !strings.Contains(got["error"].(string), "err-client-outdated") {
		t.Fatalf("work status = %v", got)
	}
	got = qrStatus(t, s, "")
	if got["authenticated"] != false || got["error"] != "" || got["qr"] != "2@abc" {
		t.Fatalf("default status = %v", got)
	}
}

func TestRegisterWhatsAppAccountLinksAndSaves(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())
	s := &Server{cfg: config.Default()}
	entry, err := s.cfg.ResolveWhatsAppAccount("assistant", "channel", "919876543210@s.whatsapp.net")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if err := s.registerWhatsAppAccount("assistant", entry); err != nil {
		t.Fatalf("register: %v", err)
	}
	if !config.IsWhatsAppAccountLinked("assistant") {
		t.Fatal("assistant not linked")
	}
	saved, err := config.LoadFrom(config.FilePath())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if a := saved.WhatsApp.Accounts["assistant"]; a == nil || a.Role != "channel" || a.OwnerJID != "919876543210@s.whatsapp.net" {
		t.Fatalf("saved accounts = %+v", saved.WhatsApp.Accounts)
	}
}
