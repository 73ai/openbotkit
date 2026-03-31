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
	"HomeTimeline":       {QueryID: "DXmgQYmIft1oLP6vMkJixw", Method: "GET"},
	"HomeLatestTimeline": {QueryID: "SFxmNKWfN9ySJcXG_tjX8g", Method: "GET"},
	"TweetDetail":        {QueryID: "iFEr5AcP121Og4wx9Yqo3w", Method: "GET"},
	"SearchTimeline":     {QueryID: "4fpceYZ6-YQCx_JSl_Cn_A", Method: "GET"},
	"UserByScreenName":   {QueryID: "ck5KkZ8t5cOmoLssopN99Q", Method: "GET"},
	"UserTweets":         {QueryID: "E8Wq-_jFSaU7hxVcuOPR9g", Method: "GET"},
	"CreateTweet":        {QueryID: "mGOM24dT4fPg08ByvrpP2A", Method: "POST"},
	"CreateRetweet":      {QueryID: "ojPdsZsimiJrUGLR1sjUtA", Method: "POST"},
	"FavoriteTweet":      {QueryID: "lI07N6Otwv1PhnEgXILM7A", Method: "POST"},
	"Notifications":      {QueryID: "l6ovGrjBwVobgU4puBCycg", Method: "GET"},
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
