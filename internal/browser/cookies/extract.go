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

// AvailableBrowsers returns browser names available on the current platform.
func AvailableBrowsers() []string {
	browsers := []string{"Chrome", "Firefox"}
	if runtime.GOOS == "darwin" {
		browsers = append([]string{"Safari"}, browsers...)
	}
	return browsers
}

// ExtractTwitterCookies tries all available browsers in order.
func ExtractTwitterCookies() (*Result, error) {
	browsers := AvailableBrowsers()
	var errs []string
	for _, name := range browsers {
		result, err := ExtractTwitterCookiesFromBrowser(name)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		return result, nil
	}
	return nil, fmt.Errorf("no X session found in any browser:\n  %s", strings.Join(errs, "\n  "))
}

// ExtractTwitterCookiesFromBrowser extracts cookies from a specific browser.
func ExtractTwitterCookiesFromBrowser(browser string) (*Result, error) {
	var fn func([]string, []string) (map[string]string, error)

	switch browser {
	case "Safari":
		if runtime.GOOS != "darwin" {
			return nil, fmt.Errorf("Safari is only available on macOS")
		}
		fn = ExtractSafariCookie
	case "Chrome":
		fn = ExtractChromeCookie
	case "Firefox":
		fn = ExtractFirefoxCookie
	default:
		return nil, fmt.Errorf("unsupported browser %q", browser)
	}

	cookies, err := fn(twitterHosts, twitterNames)
	if err != nil {
		return nil, err
	}

	authToken := cookies["auth_token"]
	if authToken == "" {
		return nil, fmt.Errorf("auth_token not found")
	}

	return &Result{
		AuthToken: authToken,
		CSRFToken: cookies["ct0"],
		Browser:   browser,
	}, nil
}
