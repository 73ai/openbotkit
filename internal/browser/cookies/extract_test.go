package cookies

import (
	"testing"
)

func TestExtractTwitterCookies_Variables(t *testing.T) {
	// Verify the package-level host/name lists are correctly configured.
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
