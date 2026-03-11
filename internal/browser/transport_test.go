package browser

import (
	"testing"
	"time"
)

func TestNewChromeTransport(t *testing.T) {
	tr := NewChromeTransport()
	if tr == nil {
		t.Fatal("NewChromeTransport returned nil")
	}
	ft, ok := tr.(*fallbackTransport)
	if !ok {
		t.Fatal("expected *fallbackTransport")
	}
	if ft.h2 == nil {
		t.Error("h2 transport is nil")
	}
	if ft.h1 == nil {
		t.Error("h1 transport is nil")
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient()
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.Jar == nil {
		t.Error("cookie jar is nil")
	}
	if c.Transport == nil {
		t.Error("transport is nil")
	}
}

func TestNewClientWithTimeout(t *testing.T) {
	c := NewClient(WithTimeout(5 * time.Second))
	if c.Timeout != 5*time.Second {
		t.Fatalf("expected 5s timeout, got %v", c.Timeout)
	}
}

func TestNewClientWithProxy(t *testing.T) {
	c := NewClient(WithProxy("http://proxy.example.com:8080"))
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	ft, ok := c.Transport.(*fallbackTransport)
	if !ok {
		t.Fatal("expected *fallbackTransport")
	}
	if ft.h1.Proxy == nil {
		t.Error("proxy not configured on h1 transport")
	}
}
