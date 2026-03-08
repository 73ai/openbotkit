package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/daemon/service"
	"github.com/priyanshujain/openbotkit/internal/skills"
	"github.com/priyanshujain/openbotkit/provider"
)

type checkResult struct {
	Name   string
	Status string // "OK", "FAIL", "WARN"
	Detail string
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the health of your obk installation",
	RunE: func(cmd *cobra.Command, args []string) error {
		var results []checkResult
		results = append(results, checkConfig()...)
		results = append(results, checkAPIKeys()...)
		results = append(results, checkGoogleOAuth()...)
		results = append(results, checkWhatsAppSession()...)
		results = append(results, checkDatabases()...)
		results = append(results, checkService())
		results = append(results, checkSkills())

		for _, r := range results {
			fmt.Fprintf(os.Stdout, "%-20s %-6s %s\n", r.Name, r.Status, r.Detail)
		}
		return nil
	},
}

func checkConfig() []checkResult {
	var results []checkResult

	_, err := os.Stat(config.FilePath())
	if err != nil {
		results = append(results, checkResult{"Config file", "FAIL", "not found: " + config.FilePath()})
		return results
	}
	results = append(results, checkResult{"Config file", "OK", config.FilePath()})

	cfg, err := config.Load()
	if err != nil {
		results = append(results, checkResult{"Config parse", "FAIL", err.Error()})
		return results
	}

	if err := cfg.RequireSetup(); err != nil {
		results = append(results, checkResult{"LLM models", "FAIL", "run 'obk setup' to configure models"})
	} else {
		results = append(results, checkResult{"LLM models", "OK", "default: " + cfg.Models.Default})
	}

	return results
}

func checkAPIKeys() []checkResult {
	cfg, err := config.Load()
	if err != nil {
		return nil
	}
	if cfg.Models == nil {
		return nil
	}

	var results []checkResult
	for name := range cfg.Models.Providers {
		providerCfg := cfg.Models.Providers[name]
		if providerCfg.AuthMethod == "vertex_ai" {
			results = append(results, checkResult{"API key (" + name + ")", "OK", "vertex_ai auth"})
			continue
		}
		envVar := provider.ProviderEnvVars[name]
		_, err := provider.ResolveAPIKey(providerCfg.APIKeyRef, envVar)
		if err != nil {
			results = append(results, checkResult{"API key (" + name + ")", "WARN", "not found"})
		} else {
			results = append(results, checkResult{"API key (" + name + ")", "OK", "resolved"})
		}
	}
	return results
}

func checkGoogleOAuth() []checkResult {
	cfg, err := config.Load()
	if err != nil {
		return nil
	}
	path := cfg.GoogleTokenDBPath()
	if _, err := os.Stat(path); err != nil {
		return []checkResult{{"Google OAuth", "WARN", "no token DB"}}
	}
	return []checkResult{{"Google OAuth", "OK", path}}
}

func checkWhatsAppSession() []checkResult {
	cfg, err := config.Load()
	if err != nil {
		return nil
	}
	path := cfg.WhatsAppSessionDBPath()
	if _, err := os.Stat(path); err != nil {
		return []checkResult{{"WhatsApp session", "WARN", "no session DB"}}
	}
	return []checkResult{{"WhatsApp session", "OK", path}}
}

func checkDatabases() []checkResult {
	cfg, err := config.Load()
	if err != nil {
		return nil
	}

	var results []checkResult
	dbs := map[string]string{
		"Gmail DB":      cfg.GmailDataDSN(),
		"WhatsApp DB":   cfg.WhatsAppDataDSN(),
		"Memory DB":     cfg.MemoryDataDSN(),
		"AppleNotes DB": cfg.AppleNotesDataDSN(),
		"Jobs DB":       cfg.JobsDBDSN(),
	}
	for name, path := range dbs {
		if _, err := os.Stat(path); err != nil {
			results = append(results, checkResult{name, "WARN", "not found"})
		} else {
			results = append(results, checkResult{name, "OK", path})
		}
	}
	return results
}

func checkService() checkResult {
	mgr, err := service.NewManager()
	if err != nil {
		return checkResult{"Service", "WARN", err.Error()}
	}
	status, err := mgr.Status()
	if err != nil {
		return checkResult{"Service", "WARN", err.Error()}
	}
	return checkResult{"Service", "OK", status}
}

func checkSkills() checkResult {
	idx, err := skills.LoadIndex()
	if err != nil {
		return checkResult{"Skills", "WARN", err.Error()}
	}
	if len(idx.Skills) == 0 {
		return checkResult{"Skills", "WARN", "no skills installed"}
	}
	return checkResult{"Skills", "OK", fmt.Sprintf("%d skills", len(idx.Skills))}
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
