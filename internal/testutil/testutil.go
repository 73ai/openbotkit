package testutil

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/73ai/openbotkit/internal/envload"
)

// LoadEnv reads a .env file and sets environment variables for the test.
// It walks up from the working directory looking for .env.
func LoadEnv(t *testing.T) {
	t.Helper()
	envload.Load(t)
}

// RequireGeminiKey loads .env and returns the Gemini API key, skipping if unavailable.
func RequireGeminiKey(t *testing.T) string {
	t.Helper()
	LoadEnv(t)
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		t.Skip("GEMINI_API_KEY not set — skipping")
	}
	return key
}

// ProviderEnabled reports whether real-LLM tests may call the named provider.
// Defaults to Gemini only; set OBK_TEST_PROVIDERS (comma separated) to opt others in.
func ProviderEnabled(name string) bool {
	list := os.Getenv("OBK_TEST_PROVIDERS")
	if list == "" {
		list = "gemini,gemini-vertex"
	}
	return slices.Contains(strings.Split(list, ","), name)
}
