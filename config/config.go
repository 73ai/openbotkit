package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Mode string

const (
	ModeLocal  Mode = "local"
	ModeRemote Mode = "remote"
	ModeServer Mode = "server"
)

type Config struct {
	Mode         Mode                `yaml:"mode,omitempty"`
	Providers    *ProvidersConfig    `yaml:"providers,omitempty"`
	Models       *ModelsConfig       `yaml:"models,omitempty"`
	Remote       *RemoteConfig       `yaml:"remote,omitempty"`
	Auth         *AuthConfig         `yaml:"auth,omitempty"`
	Channels     *ChannelsConfig     `yaml:"channels,omitempty"`
	Gmail        *GmailConfig        `yaml:"gmail,omitempty"`
	WhatsApp     *WhatsAppConfig     `yaml:"whatsapp,omitempty"`
	History      *HistoryConfig      `yaml:"history,omitempty"`
	AppleNotes   *AppleNotesConfig   `yaml:"applenotes,omitempty"`
	IMessage     *IMessageConfig     `yaml:"imessage,omitempty"`
	UserMemory   *UserMemoryConfig   `yaml:"user_memory,omitempty"`
	Daemon       *DaemonConfig       `yaml:"daemon,omitempty"`
	Usage        *UsageConfig        `yaml:"usage,omitempty"`
	Integrations *IntegrationsConfig `yaml:"integrations,omitempty"`
	WebSearch    *WebSearchConfig    `yaml:"websearch,omitempty"`
	Contacts     *ContactsConfig     `yaml:"contacts,omitempty"`
}

func (c *Config) ResolvedMode() Mode {
	if c.Mode == "" {
		return ModeLocal
	}
	return c.Mode
}

func (c *Config) IsLocal() bool  { return c.ResolvedMode() == ModeLocal }
func (c *Config) IsRemote() bool { return c.ResolvedMode() == ModeRemote }
func (c *Config) IsServer() bool { return c.ResolvedMode() == ModeServer }

type RemoteConfig struct {
	Server   string `yaml:"server,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

type AuthConfig struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

type ChannelsConfig struct {
	Telegram *TelegramConfig `yaml:"telegram,omitempty"`
}

type TelegramConfig struct {
	BotToken string `yaml:"bot_token,omitempty"`
	OwnerID  int64  `yaml:"owner_id,omitempty"`
}

// ModelsConfig configures LLM model providers and routing.
type ModelsConfig struct {
	Default   string                        `yaml:"default,omitempty"`
	Complex   string                        `yaml:"complex,omitempty"`
	Fast      string                        `yaml:"fast,omitempty"`
	Providers map[string]ModelProviderConfig `yaml:"providers,omitempty"`
}

// ModelProviderConfig holds settings for a single LLM provider.
type ModelProviderConfig struct {
	APIKeyRef     string `yaml:"api_key_ref,omitempty"`     // e.g. "keychain:obk/anthropic"
	BaseURL       string `yaml:"base_url,omitempty"`
	AuthMethod    string `yaml:"auth_method,omitempty"`     // "api_key" or "vertex_ai"
	VertexProject string `yaml:"vertex_project,omitempty"`
	VertexRegion  string `yaml:"vertex_region,omitempty"`
	VertexAccount string `yaml:"vertex_account,omitempty"`  // gcloud account email
}

type IntegrationsConfig struct {
	GWS *GWSConfig `yaml:"gws,omitempty"`
}

type GWSConfig struct {
	Enabled  bool     `yaml:"enabled,omitempty"`
	Services []string `yaml:"services,omitempty"`
}

type ProvidersConfig struct {
	Google *GoogleProviderConfig `yaml:"google,omitempty"`
}

type GoogleProviderConfig struct {
	CredentialsFile string `yaml:"credentials_file,omitempty"`
}

type DaemonConfig struct {
	GmailSyncPeriod string        `yaml:"gmail_sync_period,omitempty"` // default "15m"
	JobsStorage     StorageConfig `yaml:"jobs_storage,omitempty"`
}

type WhatsAppConfig struct {
	Storage StorageConfig `yaml:"storage,omitempty"`
}

type GmailConfig struct {
	CredentialsFile     string        `yaml:"credentials_file,omitempty"`
	DownloadAttachments bool          `yaml:"download_attachments,omitempty"`
	Storage             StorageConfig `yaml:"storage,omitempty"`
}

type HistoryConfig struct {
	Storage StorageConfig `yaml:"storage,omitempty"`
}

type UserMemoryConfig struct {
	Storage StorageConfig `yaml:"storage,omitempty"`
}

type AppleNotesConfig struct {
	Storage StorageConfig `yaml:"storage,omitempty"`
}

type UsageConfig struct {
	Storage StorageConfig `yaml:"storage,omitempty"`
}

type IMessageConfig struct {
	Storage StorageConfig `yaml:"storage,omitempty"`
}

type WebSearchConfig struct {
	Storage  StorageConfig `yaml:"storage,omitempty"`
	Proxy    string        `yaml:"proxy,omitempty"`
	Timeout  string        `yaml:"timeout,omitempty"`
	CacheTTL string        `yaml:"cache_ttl,omitempty"`
}

type ContactsConfig struct {
	Storage StorageConfig `yaml:"storage,omitempty"`
}

type StorageConfig struct {
	Driver string `yaml:"driver,omitempty"` // "sqlite" or "postgres"
	DSN    string `yaml:"dsn,omitempty"`
}

func (c *Config) SourceDataDSN(source string) (string, error) {
	switch source {
	case "gmail":
		return c.GmailDataDSN(), nil
	case "whatsapp":
		return c.WhatsAppDataDSN(), nil
	case "history":
		return c.HistoryDataDSN(), nil
	case "user_memory":
		return c.UserMemoryDataDSN(), nil
	case "applenotes":
		return c.AppleNotesDataDSN(), nil
	case "usage":
		return c.UsageDataDSN(), nil
	case "imessage":
		return c.IMessageDataDSN(), nil
	case "websearch":
		return c.WebSearchDataDSN(), nil
	case "contacts":
		return c.ContactsDataDSN(), nil
	default:
		return "", fmt.Errorf("unknown source: %q (valid: gmail, whatsapp, history, user_memory, applenotes, imessage, usage, websearch, contacts)", source)
	}
}

// RequireSetup returns an error if LLM models have not been configured.
func (c *Config) RequireSetup() error {
	if c.Models == nil || c.Models.Default == "" {
		return fmt.Errorf("setup not complete — please run 'obk setup' and configure LLM models before using this command")
	}
	return nil
}

func Load() (*Config, error) {
	return LoadFrom(FilePath())
}

func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.applyDefaults()
	return &cfg, nil
}

func (c *Config) Save() error {
	return c.SaveTo(FilePath())
}

func (c *Config) SaveTo(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

func Default() *Config {
	cfg := &Config{
		Gmail: &GmailConfig{
			Storage: StorageConfig{
				Driver: "sqlite",
			},
		},
		WhatsApp: &WhatsAppConfig{
			Storage: StorageConfig{
				Driver: "sqlite",
			},
		},
		History: &HistoryConfig{
			Storage: StorageConfig{
				Driver: "sqlite",
			},
		},
		UserMemory: &UserMemoryConfig{
			Storage: StorageConfig{
				Driver: "sqlite",
			},
		},
		AppleNotes: &AppleNotesConfig{
			Storage: StorageConfig{
				Driver: "sqlite",
			},
		},
		Usage: &UsageConfig{
			Storage: StorageConfig{
				Driver: "sqlite",
			},
		},
		IMessage: &IMessageConfig{
			Storage: StorageConfig{
				Driver: "sqlite",
			},
		},
		WebSearch: &WebSearchConfig{
			Storage: StorageConfig{
				Driver: "sqlite",
			},
		},
		Contacts: &ContactsConfig{
			Storage: StorageConfig{
				Driver: "sqlite",
			},
		},
	}
	cfg.applyDefaults()
	return cfg
}

func (c *Config) applyDefaults() {
	if c.Gmail == nil {
		c.Gmail = &GmailConfig{}
	}
	if c.Gmail.Storage.Driver == "" {
		c.Gmail.Storage.Driver = "sqlite"
	}
	if c.Gmail.CredentialsFile == "" {
		c.Gmail.CredentialsFile = filepath.Join(SourceDir("gmail"), "credentials.json")
	}
	if c.WhatsApp == nil {
		c.WhatsApp = &WhatsAppConfig{}
	}
	if c.WhatsApp.Storage.Driver == "" {
		c.WhatsApp.Storage.Driver = "sqlite"
	}
	if c.History == nil {
		c.History = &HistoryConfig{}
	}
	if c.History.Storage.Driver == "" {
		c.History.Storage.Driver = "sqlite"
	}
	if c.UserMemory == nil {
		c.UserMemory = &UserMemoryConfig{}
	}
	if c.UserMemory.Storage.Driver == "" {
		c.UserMemory.Storage.Driver = "sqlite"
	}
	if c.AppleNotes == nil {
		c.AppleNotes = &AppleNotesConfig{}
	}
	if c.AppleNotes.Storage.Driver == "" {
		c.AppleNotes.Storage.Driver = "sqlite"
	}
	if c.Usage == nil {
		c.Usage = &UsageConfig{}
	}
	if c.Usage.Storage.Driver == "" {
		c.Usage.Storage.Driver = "sqlite"
	}
	if c.IMessage == nil {
		c.IMessage = &IMessageConfig{}
	}
	if c.IMessage.Storage.Driver == "" {
		c.IMessage.Storage.Driver = "sqlite"
	}
	if c.WebSearch == nil {
		c.WebSearch = &WebSearchConfig{}
	}
	if c.WebSearch.Storage.Driver == "" {
		c.WebSearch.Storage.Driver = "sqlite"
	}
	if c.WebSearch.Timeout == "" {
		c.WebSearch.Timeout = "15s"
	}
	if c.WebSearch.CacheTTL == "" {
		c.WebSearch.CacheTTL = "15m"
	}
	if c.Contacts == nil {
		c.Contacts = &ContactsConfig{}
	}
	if c.Contacts.Storage.Driver == "" {
		c.Contacts.Storage.Driver = "sqlite"
	}
	if c.Daemon == nil {
		c.Daemon = &DaemonConfig{}
	}
	if c.Daemon.GmailSyncPeriod == "" {
		c.Daemon.GmailSyncPeriod = "15m"
	}
}

func (c *Config) GmailDataDSN() string {
	if c.Gmail.Storage.DSN != "" {
		return c.Gmail.Storage.DSN
	}
	return filepath.Join(SourceDir("gmail"), "data.db")
}

func (c *Config) WhatsAppDataDSN() string {
	if c.WhatsApp.Storage.DSN != "" {
		return c.WhatsApp.Storage.DSN
	}
	return filepath.Join(SourceDir("whatsapp"), "data.db")
}

func (c *Config) WhatsAppSessionDBPath() string {
	return filepath.Join(SourceDir("whatsapp"), "session.db")
}

func (c *Config) HistoryDataDSN() string {
	if c.History.Storage.DSN != "" {
		return c.History.Storage.DSN
	}
	return filepath.Join(SourceDir("history"), "data.db")
}

func (c *Config) UserMemoryDataDSN() string {
	if c.UserMemory.Storage.DSN != "" {
		return c.UserMemory.Storage.DSN
	}
	return filepath.Join(SourceDir("user_memory"), "data.db")
}

func (c *Config) UsageDataDSN() string {
	if c.Usage.Storage.DSN != "" {
		return c.Usage.Storage.DSN
	}
	return filepath.Join(SourceDir("usage"), "data.db")
}

func (c *Config) AppleNotesDataDSN() string {
	if c.AppleNotes.Storage.DSN != "" {
		return c.AppleNotes.Storage.DSN
	}
	return filepath.Join(SourceDir("applenotes"), "data.db")
}

func (c *Config) WebSearchDataDSN() string {
	if c.WebSearch.Storage.DSN != "" {
		return c.WebSearch.Storage.DSN
	}
	return filepath.Join(SourceDir("websearch"), "data.db")
}

func (c *Config) IMessageDataDSN() string {
	if c.IMessage.Storage.DSN != "" {
		return c.IMessage.Storage.DSN
	}
	return filepath.Join(SourceDir("imessage"), "data.db")
}

func (c *Config) ContactsDataDSN() string {
	if c.Contacts.Storage.DSN != "" {
		return c.Contacts.Storage.DSN
	}
	return filepath.Join(SourceDir("contacts"), "data.db")
}

func (c *Config) JobsDBDSN() string {
	if c.Daemon.JobsStorage.DSN != "" {
		return c.Daemon.JobsStorage.DSN
	}
	return JobsDBPath()
}

// GoogleCredentialsFile returns the credentials file path, checking the new
// providers config first, falling back to the legacy gmail config.
func (c *Config) GoogleCredentialsFile() string {
	if c.Providers != nil && c.Providers.Google != nil && c.Providers.Google.CredentialsFile != "" {
		return c.Providers.Google.CredentialsFile
	}
	if c.Gmail != nil && c.Gmail.CredentialsFile != "" {
		return c.Gmail.CredentialsFile
	}
	return filepath.Join(ProviderDir("google"), "credentials.json")
}

// GoogleTokenDBPath always points to the new provider location.
func (c *Config) GoogleTokenDBPath() string {
	return filepath.Join(ProviderDir("google"), "tokens.db")
}

