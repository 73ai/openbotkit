package server

import (
	"html"
	"net/http"
	"strings"
)

func (s *Server) handleAuthRedirect(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		http.Error(w, "missing url parameter", http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(rawURL, "https://accounts.google.com/") {
		http.Error(w, "url must be a Google OAuth URL", http.StatusBadRequest)
		return
	}

	safeURL := html.EscapeString(rawURL)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Continue to Google</title>
<style>
body{font-family:-apple-system,system-ui,sans-serif;display:flex;justify-content:center;align-items:center;min-height:100vh;margin:0;background:#f5f5f5}
.card{background:#fff;border-radius:12px;padding:2rem;text-align:center;box-shadow:0 2px 8px rgba(0,0,0,.1);max-width:320px}
a.btn{display:inline-block;margin-top:1rem;padding:.75rem 1.5rem;background:#4285f4;color:#fff;text-decoration:none;border-radius:8px;font-size:1.1rem}
</style></head>
<body>
<div class="card">
<p>Tap the button below to sign in with your Google account.</p>
<a class="btn" id="auth-link" href="` + safeURL + `" target="_blank" rel="noopener" onclick="return openInBrowser(event)">Continue with Google</a>
</div>
<script src="https://telegram.org/js/telegram-web-app.js"></script>
<script>
function openInBrowser(e) {
  var tg = window.Telegram && window.Telegram.WebApp;
  if (tg && tg.openLink) {
    e.preventDefault();
    tg.openLink(document.getElementById('auth-link').href);
    tg.close();
    return false;
  }
  return true;
}
(function(){
  var tg = window.Telegram && window.Telegram.WebApp;
  if (!tg && !window.TelegramWebviewProxy && !/Telegram/i.test(navigator.userAgent)) {
    window.location.replace(document.getElementById('auth-link').href);
  }
})();
</script>
</body></html>`))
}
