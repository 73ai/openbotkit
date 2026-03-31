package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/73ai/openbotkit/source/twitter/client"
	"gopkg.in/yaml.v3"
)

var (
	chunkMapRe = regexp.MustCompile(`"ondemand\.s":"([a-f0-9]+)"`)
	queryIDRe  = regexp.MustCompile(`queryId:"([^"]+)".+?operationName:"([^"]+)"`)
	queryIDRe2 = regexp.MustCompile(`operationName:"([^"]+)".+?queryId:"([^"]+)"`)
)

var knownOperations = map[string]string{
	"HomeTimeline":       "GET",
	"HomeLatestTimeline": "GET",
	"TweetDetail":        "GET",
	"SearchTimeline":     "GET",
	"UserByScreenName":   "GET",
	"UserTweets":         "GET",
	"CreateTweet":        "POST",
	"CreateRetweet":      "POST",
	"FavoriteTweet":      "POST",
	"Notifications":      "GET",
}

func extractEndpoints(httpClient *http.Client) (map[string]client.Endpoint, error) {
	mainJS, err := fetchMainPage(httpClient)
	if err != nil {
		return nil, fmt.Errorf("fetch main page: %w", err)
	}

	chunkHashes := extractChunkHashes(mainJS)
	if len(chunkHashes) == 0 {
		return nil, fmt.Errorf("no chunk hashes found in main page JS")
	}

	endpoints := make(map[string]client.Endpoint)

	for _, hash := range chunkHashes {
		jsURL := fmt.Sprintf("https://abs.twimg.com/responsive-web/client-web/ondemand.s.%s.js", hash)
		body, err := fetchURL(httpClient, jsURL)
		if err != nil {
			continue
		}

		found := extractQueryIDs(body)
		for op, qid := range found {
			if method, ok := knownOperations[op]; ok {
				endpoints[op] = client.Endpoint{QueryID: qid, Method: method}
			}
		}
	}

	return endpoints, nil
}

func fetchMainPage(httpClient *http.Client) (string, error) {
	body, err := fetchURL(httpClient, "https://x.com")
	if err != nil {
		return "", err
	}

	scriptRe := regexp.MustCompile(`src="(https://abs\.twimg\.com/responsive-web/client-web[^"]+\.js)"`)
	matches := scriptRe.FindAllStringSubmatch(body, -1)

	var allJS strings.Builder
	allJS.WriteString(body)

	for _, m := range matches {
		js, err := fetchURL(httpClient, m[1])
		if err != nil {
			continue
		}
		allJS.WriteString(js)
	}

	return allJS.String(), nil
}

func fetchURL(httpClient *http.Client, url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func extractChunkHashes(js string) []string {
	matches := chunkMapRe.FindAllStringSubmatch(js, -1)
	seen := make(map[string]bool)
	var hashes []string
	for _, m := range matches {
		if !seen[m[1]] {
			seen[m[1]] = true
			hashes = append(hashes, m[1])
		}
	}
	return hashes
}

func extractQueryIDs(js string) map[string]string {
	result := make(map[string]string)

	for _, m := range queryIDRe.FindAllStringSubmatch(js, -1) {
		result[m[2]] = m[1]
	}
	for _, m := range queryIDRe2.FindAllStringSubmatch(js, -1) {
		result[m[1]] = m[2]
	}

	return result
}

func endpointsToYAML(endpoints map[string]client.Endpoint) ([]byte, error) {
	ordered := make([]string, 0, len(endpoints))
	for k := range endpoints {
		ordered = append(ordered, k)
	}
	sort.Strings(ordered)

	type yamlEndpoint struct {
		QueryID string `yaml:"query_id"`
		Method  string `yaml:"method"`
	}
	m := make(map[string]yamlEndpoint, len(endpoints))
	for _, k := range ordered {
		ep := endpoints[k]
		m[k] = yamlEndpoint{QueryID: ep.QueryID, Method: ep.Method}
	}

	return yaml.Marshal(m)
}
