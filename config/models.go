package config

// ModelInfo describes a model available for profile configuration.
type ModelInfo struct {
	Provider       string
	ID             string
	Label          string
	ContextWindow  int
	RecommendedFor []string // "default", "complex", "fast", "nano"
}
