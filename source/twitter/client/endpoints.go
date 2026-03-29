package client

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Endpoint struct {
	QueryID string `yaml:"query_id"`
	Method  string `yaml:"method"`
}

var defaultEndpoints = map[string]Endpoint{
	"HomeTimeline":       {QueryID: "HCosKfLNW1AcOo3la3mMgg", Method: "GET"},
	"HomeLatestTimeline": {QueryID: "DiTkXJgAqBPytnwAriHOhw", Method: "GET"},
	"TweetDetail":        {QueryID: "nBS-WpgA6ZG0CyNHD517JQ", Method: "GET"},
	"SearchTimeline":     {QueryID: "gkjsKepM6gl_HmFWoWKfgg", Method: "GET"},
	"UserByScreenName":   {QueryID: "xmU6X_CKVnQ5lSrCbAmJsg", Method: "GET"},
	"UserTweets":         {QueryID: "E3opETHurmVJflFsUBVuUQ", Method: "GET"},
	"CreateTweet":        {QueryID: "a1p9RWpkYKBjWv_I3WzS-A", Method: "POST"},
	"CreateRetweet":      {QueryID: "ojPdsZsimiJrUGLR1sjVsA", Method: "POST"},
	"FavoriteTweet":      {QueryID: "lI07N6Otwv1PhnEgXILM7A", Method: "POST"},
	"Notifications":      {QueryID: "PsAvrHFgRwVBSRYAulPemA", Method: "GET"},
}

func DefaultEndpointsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".obk", "twitter", "endpoints.yaml")
	}
	return filepath.Join(home, ".obk", "twitter", "endpoints.yaml")
}

func LoadEndpoints(path string) (map[string]Endpoint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultEndpoints(), nil
		}
		return nil, fmt.Errorf("read endpoints file: %w", err)
	}

	var endpoints map[string]Endpoint
	if err := yaml.Unmarshal(data, &endpoints); err != nil {
		return nil, fmt.Errorf("parse endpoints file: %w", err)
	}

	if len(endpoints) == 0 {
		return nil, fmt.Errorf("endpoints file is empty")
	}
	return endpoints, nil
}

func DefaultEndpoints() map[string]Endpoint {
	m := make(map[string]Endpoint, len(defaultEndpoints))
	for k, v := range defaultEndpoints {
		m[k] = v
	}
	return m
}
