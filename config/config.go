package config

import (
	"fmt"
	"os"
	"time"

	"github.com/priyanshujain/reimbursement/parser"
	"github.com/priyanshujain/reimbursement/recon"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Accounts         []string            `yaml:"accounts"`
	AfterDate        string              `yaml:"after_date"`
	Services         []ServiceConfig     `yaml:"services"`
	FinancialSources []FinancialSource   `yaml:"financial_sources"`
	DashboardSources []DashboardSource   `yaml:"dashboard_sources"`
	PDFPasswords     map[string]string   `yaml:"pdf_passwords"`
}

type ServiceConfig struct {
	Name             string            `yaml:"name"`
	AfterDate        string            `yaml:"after_date"`
	EmailParser      string            `yaml:"email_parser"`
	DashboardInvoice string            `yaml:"dashboard_invoice"`
	OfflineDir       string            `yaml:"offline_dir"`
	DestPatterns     []string          `yaml:"dest_patterns"`
	EmailFroms       []string          `yaml:"email_froms"`
	EmailSubjects    []string          `yaml:"email_subjects"`
	ExcludeSubjects  []string          `yaml:"exclude_subjects"`
	Surcharges       []SurchargeConfig `yaml:"surcharges"`
	Optional         bool              `yaml:"optional"`
}

type SurchargeConfig struct {
	Pattern    string  `yaml:"pattern"`
	Percentage float64 `yaml:"percentage"`
	MaxDays    int     `yaml:"max_days"`
	GSTRate    float64 `yaml:"gst_rate"`
}

type FinancialSource struct {
	Name     string `yaml:"name"`
	Parser   string `yaml:"parser"`
	Glob     string `yaml:"glob"`
	Password string `yaml:"password"` // key into PDFPasswords
}

type DashboardSource struct {
	Name   string `yaml:"name"`
	Parser string `yaml:"parser"`
	Glob   string `yaml:"glob"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// BuildServices resolves service configs into recon.Service values using the parser registry.
func BuildServices(cfg *Config) ([]recon.Service, error) {
	services := make([]recon.Service, 0, len(cfg.Services))

	for _, sc := range cfg.Services {
		svc := recon.Service{
			Name:             sc.Name,
			DestPatterns:     sc.DestPatterns,
			EmailFroms:       sc.EmailFroms,
			EmailSubjects:    sc.EmailSubjects,
			ExcludeSubject:   sc.ExcludeSubjects,
			DashboardInvoice: sc.DashboardInvoice,
			OfflineDir:       sc.OfflineDir,
			Optional:         sc.Optional,
		}

		// Parse after_date
		if sc.AfterDate != "" {
			t, err := time.Parse("2006/01/02", sc.AfterDate)
			if err != nil {
				return nil, fmt.Errorf("service %q: parse after_date %q: %w", sc.Name, sc.AfterDate, err)
			}
			svc.AfterDate = t
		}

		// Resolve email parser
		if sc.EmailParser != "" {
			p, err := parser.GetEmailParser(sc.EmailParser)
			if err != nil {
				return nil, fmt.Errorf("service %q: %w", sc.Name, err)
			}
			svc.EmailParser = p
		}

		// Convert surcharge configs
		for _, sur := range sc.Surcharges {
			svc.Surcharges = append(svc.Surcharges, recon.SurchargeRule{
				Pattern:    sur.Pattern,
				Percentage: sur.Percentage,
				MaxDays:    sur.MaxDays,
				GSTRate:    sur.GSTRate,
			})
		}

		services = append(services, svc)
	}

	return services, nil
}

// FetchSourcesFromServices deduplicates EmailFroms across all services.
func FetchSourcesFromServices(services []recon.Service) []string {
	seen := make(map[string]bool)
	var sources []string
	for _, svc := range services {
		for _, from := range svc.EmailFroms {
			if !seen[from] {
				seen[from] = true
				sources = append(sources, from)
			}
		}
	}
	return sources
}
