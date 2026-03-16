package websearch

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
)

type backendInfo struct {
	Name     string `json:"name"`
	Priority int    `json:"priority"`
	News     bool   `json:"news"`
}

var backendsCmd = &cobra.Command{
	Use:     "backends",
	Short:   "List available search backends",
	Example: `  obk websearch backends`,
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		backends := []backendInfo{
			{Name: "duckduckgo", Priority: 1, News: true},
			{Name: "brave", Priority: 1, News: false},
			{Name: "mojeek", Priority: 1, News: false},
			{Name: "yahoo", Priority: 1, News: true},
			{Name: "yandex", Priority: 1, News: false},
			{Name: "google", Priority: 0, News: false},
			{Name: "wikipedia", Priority: 2, News: false},
			{Name: "bing", Priority: 0, News: false},
		}
		return json.NewEncoder(os.Stdout).Encode(backends)
	},
}
