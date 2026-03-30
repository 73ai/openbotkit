package cookies

import (
	"runtime"
	"testing"
)

func TestExtractTwitterCookies_Variables(t *testing.T) {
	if len(twitterHosts) != 4 {
		t.Errorf("twitterHosts has %d entries, want 4", len(twitterHosts))
	}
	if len(twitterNames) != 2 {
		t.Errorf("twitterNames has %d entries, want 2", len(twitterNames))
	}

	hostSet := make(map[string]bool)
	for _, h := range twitterHosts {
		hostSet[h] = true
	}
	for _, expected := range []string{".x.com", "x.com", ".twitter.com", "twitter.com"} {
		if !hostSet[expected] {
			t.Errorf("twitterHosts missing %q", expected)
		}
	}

	nameSet := make(map[string]bool)
	for _, n := range twitterNames {
		nameSet[n] = true
	}
	for _, expected := range []string{"auth_token", "ct0"} {
		if !nameSet[expected] {
			t.Errorf("twitterNames missing %q", expected)
		}
	}
}

func TestResult_Fields(t *testing.T) {
	r := &Result{
		AuthToken: "abc",
		CSRFToken: "def",
		Browser:   "Safari",
	}
	if r.AuthToken != "abc" {
		t.Errorf("AuthToken = %q", r.AuthToken)
	}
	if r.CSRFToken != "def" {
		t.Errorf("CSRFToken = %q", r.CSRFToken)
	}
	if r.Browser != "Safari" {
		t.Errorf("Browser = %q", r.Browser)
	}
}

func TestAvailableBrowsers(t *testing.T) {
	browsers := AvailableBrowsers()

	has := func(name string) bool {
		for _, b := range browsers {
			if b == name {
				return true
			}
		}
		return false
	}

	if !has("Chrome") {
		t.Error("Chrome should always be available")
	}
	if !has("Firefox") {
		t.Error("Firefox should always be available")
	}
	if runtime.GOOS == "darwin" && !has("Safari") {
		t.Error("Safari should be available on macOS")
	}
	if runtime.GOOS != "darwin" && has("Safari") {
		t.Error("Safari should not be available on non-macOS")
	}
}

func TestExtractTwitterCookiesFromBrowser_UnsupportedBrowser(t *testing.T) {
	_, err := ExtractTwitterCookiesFromBrowser("Netscape")
	if err == nil {
		t.Fatal("expected error for unsupported browser")
	}
}

func TestExtractTwitterCookiesFromBrowser_SafariOnNonDarwin(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("test only applies on non-macOS")
	}
	_, err := ExtractTwitterCookiesFromBrowser("Safari")
	if err == nil {
		t.Fatal("expected error for Safari on non-macOS")
	}
}
