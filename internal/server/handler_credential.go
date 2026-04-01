package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"sync"
	"time"

	"github.com/73ai/openbotkit/provider"
)

type credentialToken struct {
	Label     string
	KeyRef    string
	ExpiresAt time.Time
	Used      bool
}

type credentialTokenStore struct {
	mu     sync.Mutex
	tokens map[string]*credentialToken
}

func newCredentialTokenStore() *credentialTokenStore {
	return &credentialTokenStore{tokens: make(map[string]*credentialToken)}
}

func (s *credentialTokenStore) create(label, keyRef string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(b)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token] = &credentialToken{
		Label:     label,
		KeyRef:    keyRef,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	return token, nil
}

func (s *credentialTokenStore) get(token string) (*credentialToken, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ct, ok := s.tokens[token]
	if !ok {
		return nil, false
	}
	if ct.Used || time.Now().After(ct.ExpiresAt) {
		return nil, false
	}
	return ct, true
}

func (s *credentialTokenStore) invalidate(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ct, ok := s.tokens[token]; ok {
		ct.Used = true
	}
}

// handleCredentialRequest creates a one-time credential submission token.
// POST /api/credential/request
// Body: {"label": "Anthropic API Key", "key_ref": "keychain:obk/anthropic"}
func (s *Server) handleCredentialRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Label  string `json:"label"`
		KeyRef string `json:"key_ref"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Label == "" || req.KeyRef == "" {
		http.Error(w, "label and key_ref are required", http.StatusBadRequest)
		return
	}

	token, err := s.credTokens.create(req.Label, req.KeyRef)
	if err != nil {
		http.Error(w, "failed to create token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
		"url":   "/credential/" + token,
	})
}

// handleCredentialForm renders the HTML form for credential input.
// GET /credential/{token}
func (s *Server) handleCredentialForm(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	ct, ok := s.credTokens.get(token)
	if !ok {
		http.Error(w, "invalid or expired token", http.StatusNotFound)
		return
	}

	safeLabel := html.EscapeString(ct.Label)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Enter Credential</title>
<style>
body{font-family:-apple-system,system-ui,sans-serif;display:flex;justify-content:center;align-items:center;min-height:100vh;margin:0;background:#f5f5f5}
.card{background:#fff;border-radius:12px;padding:2rem;text-align:center;box-shadow:0 2px 8px rgba(0,0,0,.1);max-width:360px;width:100%%}
input[type=password]{width:100%%;padding:.75rem;border:1px solid #ddd;border-radius:8px;font-size:1rem;margin:.75rem 0;box-sizing:border-box}
button{padding:.75rem 1.5rem;background:#4285f4;color:#fff;border:none;border-radius:8px;font-size:1.1rem;cursor:pointer;width:100%%}
button:hover{background:#3367d6}
.label{font-weight:600;font-size:1.1rem;margin-bottom:.5rem}
</style></head>
<body>
<div class="card">
<div class="label">%s</div>
<form method="POST" action="/credential/%s">
<input type="password" name="value" placeholder="Paste your key here" required autofocus>
<button type="submit">Save</button>
</form>
</div>
</body></html>`, safeLabel, html.EscapeString(token))
}

// handleCredentialSubmit stores the credential and invalidates the token.
// POST /credential/{token}
func (s *Server) handleCredentialSubmit(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	ct, ok := s.credTokens.get(token)
	if !ok {
		http.Error(w, "invalid or expired token", http.StatusNotFound)
		return
	}

	value := r.FormValue("value")
	if value == "" {
		http.Error(w, "value is required", http.StatusBadRequest)
		return
	}

	if err := provider.StoreCredential(ct.KeyRef, value); err != nil {
		http.Error(w, "failed to store credential", http.StatusInternalServerError)
		return
	}

	s.credTokens.invalidate(token)

	safeLabel := html.EscapeString(ct.Label)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Credential Saved</title>
<style>
body{font-family:-apple-system,system-ui,sans-serif;display:flex;justify-content:center;align-items:center;min-height:100vh;margin:0;background:#f5f5f5}
.card{background:#fff;border-radius:12px;padding:2rem;text-align:center;box-shadow:0 2px 8px rgba(0,0,0,.1);max-width:320px}
.check{font-size:3rem;margin-bottom:.5rem}
</style></head>
<body>
<div class="card">
<div class="check">✓</div>
<p><strong>%s</strong> saved successfully.</p>
<p>You can close this tab.</p>
</div>
<script src="https://telegram.org/js/telegram-web-app.js"></script>
<script>
(function(){
  var tg = window.Telegram && window.Telegram.WebApp;
  if (tg && tg.close) { setTimeout(function(){ tg.close(); }, 1500); }
})();
</script>
</body></html>`, safeLabel)
}
