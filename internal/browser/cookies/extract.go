package cookies

import (
	"fmt"
	"runtime"
	"strings"
)

var (
	twitterHosts = []string{".x.com", "x.com", ".twitter.com", "twitter.com"}
	twitterNames = []string{"auth_token", "ct0"}
)

type Result struct {
	AuthToken string
	CSRFToken string
	Browser   string
}

func ExtractTwitterCookies() (*Result, error) {
	type extractor struct {
		name string
		fn   func([]string, []string) (map[string]string, error)
	}

	extractors := []extractor{
		{"Chrome", ExtractChromeCookie},
		{"Firefox", ExtractFirefoxCookie},
	}

	if runtime.GOOS == "darwin" {
		extractors = append([]extractor{{"Safari", ExtractSafariCookie}}, extractors...)
	}

	var errs []string
	for _, ext := range extractors {
		cookies, err := ext.fn(twitterHosts, twitterNames)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", ext.name, err))
			continue
		}

		authToken := cookies["auth_token"]
		if authToken == "" {
			errs = append(errs, fmt.Sprintf("%s: auth_token not found", ext.name))
			continue
		}

		return &Result{
			AuthToken: authToken,
			CSRFToken: cookies["ct0"],
			Browser:   ext.name,
		}, nil
	}

	return nil, fmt.Errorf("no X session found in any browser:\n  %s", strings.Join(errs, "\n  "))
}
