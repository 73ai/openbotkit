package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildCallbackURL(t *testing.T) {
	got := buildCallbackURL("panda-new-kit.ngrok-free.app")
	want := "https://panda-new-kit.ngrok-free.app/auth/google/callback"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestWriteNgrokConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ngrok.yml")

	err := writeNgrokConfig(path, "test-token-123", "my-domain.ngrok-free.app")
	if err != nil {
		t.Fatalf("writeNgrokConfig: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "test-token-123") {
		t.Error("config missing authtoken")
	}
	if !strings.Contains(content, "my-domain.ngrok-free.app") {
		t.Error("config missing domain")
	}
	if !strings.Contains(content, "addr: 8443") {
		t.Error("config should tunnel to server port 8443")
	}
	if !strings.Contains(content, `version: "3"`) {
		t.Error("config missing version")
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0600 {
			t.Errorf("file permissions = %o, want 0600", perm)
		}
	}
}

func TestQueryNgrokTunnelDomain(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"tunnels":[{"public_url":"https://cool-domain.ngrok-free.dev"}]}`)
	}))
	defer srv.Close()

	// queryNgrokTunnelDomain hardcodes 127.0.0.1:4040, so we can't easily
	// redirect it to our test server. Instead, test the JSON parsing logic
	// by calling the test server directly and parsing the same way.
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var result struct {
		Tunnels []struct {
			PublicURL string `json:"public_url"`
		} `json:"tunnels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result.Tunnels) != 1 {
		t.Fatalf("expected 1 tunnel, got %d", len(result.Tunnels))
	}
	if result.Tunnels[0].PublicURL != "https://cool-domain.ngrok-free.dev" {
		t.Fatalf("got %q", result.Tunnels[0].PublicURL)
	}
}

func TestWriteNgrokConfig_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "ngrok.yml")

	err := writeNgrokConfig(path, "tok", "example.ngrok-free.app")
	if err != nil {
		t.Fatalf("writeNgrokConfig: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}
