package websearch

import (
	"context"
	"net/http"
	"time"

	"github.com/priyanshujain/openbotkit/internal/browser"
	"github.com/priyanshujain/openbotkit/source"
	"github.com/priyanshujain/openbotkit/store"
)

const chromeUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

type WebSearch struct {
	cfg      Config
	skipSSRF bool // for testing only
}

func New(cfg Config) *WebSearch {
	return &WebSearch{cfg: cfg}
}

func (w *WebSearch) Name() string {
	return "websearch"
}

func (w *WebSearch) Status(_ context.Context, db *store.DB) (*source.Status, error) {
	st := &source.Status{Connected: true}
	if db != nil {
		var count int64
		err := db.QueryRow("SELECT COUNT(*) FROM search_history").Scan(&count)
		if err == nil {
			st.ItemCount = count
		}
	}
	return st, nil
}

func (w *WebSearch) httpClient() *http.Client {
	var opts []browser.ClientOption

	if w.cfg.WebSearch != nil {
		if w.cfg.WebSearch.Timeout != "" {
			if d, err := time.ParseDuration(w.cfg.WebSearch.Timeout); err == nil {
				opts = append(opts, browser.WithTimeout(d))
			}
		}
		if w.cfg.WebSearch.Proxy != "" {
			opts = append(opts, browser.WithProxy(w.cfg.WebSearch.Proxy))
		}
	}

	return browser.NewClient(opts...)
}
