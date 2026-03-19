package provider

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CachedModelList stores the model catalog for a provider with verification state.
type CachedModelList struct {
	Provider       string                   `json:"provider"`
	Models         []AvailableModel         `json:"models"`
	FetchedAt      time.Time                `json:"fetched_at"`
	VerifiedModels map[string]VerifiedModel `json:"verified_models,omitempty"`
}

// VerifiedModel records that a model was verified with a specific API key.
type VerifiedModel struct {
	KeyHash    string    `json:"key_hash"`
	VerifiedAt time.Time `json:"verified_at"`
}

// ModelCache manages cached model lists per provider in JSON files.
type ModelCache struct {
	dir string
}

// NewModelCache creates a cache that stores files in the given directory.
func NewModelCache(dir string) *ModelCache {
	return &ModelCache{dir: dir}
}

// Load reads the cached model list for a provider.
func (c *ModelCache) Load(provider string) (*CachedModelList, error) {
	data, err := os.ReadFile(c.path(provider))
	if err != nil {
		return nil, err
	}
	var list CachedModelList
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return &list, nil
}

// Save writes the model list to the cache file.
func (c *ModelCache) Save(provider string, list *CachedModelList) error {
	if err := os.MkdirAll(c.dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path(provider), data, 0600)
}

// IsStale returns true if the cache file doesn't exist or is older than ttl.
func (c *ModelCache) IsStale(provider string, ttl time.Duration) bool {
	info, err := os.Stat(c.path(provider))
	if err != nil {
		return true
	}
	return time.Since(info.ModTime()) > ttl
}

// IsModelVerified returns true if the model was verified with the given API key.
func (c *ModelCache) IsModelVerified(provider, modelID, apiKey string) bool {
	list, err := c.Load(provider)
	if err != nil {
		return false
	}
	if list.VerifiedModels == nil {
		return false
	}
	v, ok := list.VerifiedModels[modelID]
	if !ok {
		return false
	}
	return v.KeyHash == hashKey(apiKey)
}

// MarkModelVerified records that a model was verified with the given API key.
func (c *ModelCache) MarkModelVerified(provider, modelID, apiKey string) error {
	list, err := c.Load(provider)
	if err != nil {
		list = &CachedModelList{Provider: provider}
	}
	if list.VerifiedModels == nil {
		list.VerifiedModels = make(map[string]VerifiedModel)
	}
	list.VerifiedModels[modelID] = VerifiedModel{
		KeyHash:    hashKey(apiKey),
		VerifiedAt: time.Now(),
	}
	return c.Save(provider, list)
}

func (c *ModelCache) path(provider string) string {
	return filepath.Join(c.dir, provider+".json")
}

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", h)
}
