package finance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// mockYahoo sets up a test server mimicking Yahoo Finance's session flow:
// 1. /cookie → sets A3 cookie
// 2. /crumb  → returns crumb string (requires cookie)
// 3. /quote  → returns quote JSON (requires crumb)
func mockYahoo(t *testing.T, quotes []Quote) (*httptest.Server, *Client) {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/cookie", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "A3", Value: "test-cookie"})
		w.WriteHeader(http.StatusNotFound)
	})

	mux.HandleFunc("/crumb", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("A3")
		if err != nil || cookie.Value == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write([]byte("test-crumb"))
	})

	mux.HandleFunc("/quote", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("crumb") != "test-crumb" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		resp := struct {
			QuoteResponse struct {
				Result []Quote `json:"result"`
				Error  *string `json:"error"`
			} `json:"quoteResponse"`
		}{}
		resp.QuoteResponse.Result = quotes
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	srv := httptest.NewServer(mux)
	client := newTestClient(srv)
	return srv, client
}

func newTestClient(srv *httptest.Server) *Client {
	httpClient := srv.Client()
	jar, _ := cookiejar.New(nil)
	httpClient.Jar = jar
	return &Client{
		http:      httpClient,
		cookieURL: srv.URL + "/cookie",
		crumbURL:  srv.URL + "/crumb",
		quoteURL:  srv.URL + "/quote",
	}
}

func TestNewSession(t *testing.T) {
	srv, client := mockYahoo(t, nil)
	defer srv.Close()

	err := client.initSession(context.Background())
	if err != nil {
		t.Fatalf("initSession: %v", err)
	}
	if client.crumb != "test-crumb" {
		t.Errorf("crumb = %q, want %q", client.crumb, "test-crumb")
	}
	if !client.inited {
		t.Error("client not marked as inited")
	}
}

func TestQuoteStock(t *testing.T) {
	want := Quote{
		Symbol:                     "AAPL",
		ShortName:                  "Apple Inc.",
		QuoteType:                  "EQUITY",
		Currency:                   "USD",
		Exchange:                   "NMS",
		RegularMarketPrice:         260.83,
		RegularMarketChange:        0.95,
		RegularMarketChangePercent: 0.37,
		RegularMarketDayHigh:       262.0,
		RegularMarketDayLow:        259.0,
		RegularMarketVolume:        45000000,
		MarketState:                "REGULAR",
	}

	srv, client := mockYahoo(t, []Quote{want})
	defer srv.Close()

	quotes, err := client.Quote(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Quote: %v", err)
	}
	if len(quotes) != 1 {
		t.Fatalf("got %d quotes, want 1", len(quotes))
	}
	got := quotes[0]
	if got.Symbol != want.Symbol {
		t.Errorf("Symbol = %q, want %q", got.Symbol, want.Symbol)
	}
	if got.RegularMarketPrice != want.RegularMarketPrice {
		t.Errorf("Price = %v, want %v", got.RegularMarketPrice, want.RegularMarketPrice)
	}
	if got.ShortName != want.ShortName {
		t.Errorf("ShortName = %q, want %q", got.ShortName, want.ShortName)
	}
}

func TestQuoteForex(t *testing.T) {
	want := Quote{
		Symbol:                     "USDINR=X",
		ShortName:                  "USD/INR",
		QuoteType:                  "CURRENCY",
		Currency:                   "INR",
		RegularMarketPrice:         91.85,
		RegularMarketChange:        -0.09,
		RegularMarketChangePercent: -0.10,
	}

	srv, client := mockYahoo(t, []Quote{want})
	defer srv.Close()

	quotes, err := client.Quote(context.Background(), "USDINR=X")
	if err != nil {
		t.Fatalf("Quote: %v", err)
	}
	if len(quotes) != 1 {
		t.Fatalf("got %d quotes, want 1", len(quotes))
	}
	if quotes[0].QuoteType != "CURRENCY" {
		t.Errorf("QuoteType = %q, want CURRENCY", quotes[0].QuoteType)
	}
	if quotes[0].RegularMarketPrice != 91.85 {
		t.Errorf("Price = %v, want 91.85", quotes[0].RegularMarketPrice)
	}
}

func TestQuoteMultiSymbol(t *testing.T) {
	symbols := []Quote{
		{Symbol: "AAPL", RegularMarketPrice: 260.0},
		{Symbol: "GOOGL", RegularMarketPrice: 307.0},
		{Symbol: "MSFT", RegularMarketPrice: 420.0},
	}

	srv, client := mockYahoo(t, symbols)
	defer srv.Close()

	quotes, err := client.Quote(context.Background(), "AAPL", "GOOGL", "MSFT")
	if err != nil {
		t.Fatalf("Quote: %v", err)
	}
	if len(quotes) != 3 {
		t.Fatalf("got %d quotes, want 3", len(quotes))
	}
	for i, q := range quotes {
		if q.Symbol != symbols[i].Symbol {
			t.Errorf("quotes[%d].Symbol = %q, want %q", i, q.Symbol, symbols[i].Symbol)
		}
	}
}

func TestQuoteInvalidSymbol(t *testing.T) {
	srv, client := mockYahoo(t, []Quote{})
	defer srv.Close()

	quotes, err := client.Quote(context.Background(), "XYZNOTREAL")
	if err != nil {
		t.Fatalf("Quote: %v", err)
	}
	if len(quotes) != 0 {
		t.Errorf("got %d quotes, want 0 for invalid symbol", len(quotes))
	}
}

func TestQuoteMalformedJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cookie", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "A3", Value: "test-cookie"})
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/crumb", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test-crumb"))
	})
	mux.HandleFunc("/quote", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json at all{{{"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(srv)
	_, err := client.Quote(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("error = %q, want it to mention decoding", err.Error())
	}
}

func TestSessionRefreshOn401(t *testing.T) {
	var sessionInits atomic.Int32
	var quoteRequests atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("/cookie", func(w http.ResponseWriter, r *http.Request) {
		sessionInits.Add(1)
		http.SetCookie(w, &http.Cookie{Name: "A3", Value: "test-cookie"})
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/crumb", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test-crumb"))
	})
	mux.HandleFunc("/quote", func(w http.ResponseWriter, r *http.Request) {
		n := quoteRequests.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		resp := struct {
			QuoteResponse struct {
				Result []Quote `json:"result"`
				Error  *string `json:"error"`
			} `json:"quoteResponse"`
		}{}
		resp.QuoteResponse.Result = []Quote{{Symbol: "AAPL", RegularMarketPrice: 260.0}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(srv)
	quotes, err := client.Quote(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Quote: %v", err)
	}
	if len(quotes) != 1 {
		t.Fatalf("got %d quotes, want 1", len(quotes))
	}
	// Initial session + refresh = 2 session inits
	if n := sessionInits.Load(); n != 2 {
		t.Errorf("session inits = %d, want 2", n)
	}
}

func TestCrumbEndpointFails(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cookie", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "A3", Value: "test-cookie"})
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/crumb", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(srv)
	_, err := client.Quote(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("expected error when crumb endpoint returns 429")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error = %q, want it to mention 429", err.Error())
	}
}

func TestQuote429(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cookie", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "A3", Value: "test-cookie"})
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/crumb", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test-crumb"))
	})
	mux.HandleFunc("/quote", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(srv)
	_, err := client.Quote(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("expected error for 429")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error = %q, want it to mention 429", err.Error())
	}
}

func TestQuoteTimeout(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cookie", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "A3", Value: "test-cookie"})
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/crumb", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test-crumb"))
	})
	mux.HandleFunc("/quote", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte("{}"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newTestClient(srv)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.Quote(ctx, "AAPL")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
