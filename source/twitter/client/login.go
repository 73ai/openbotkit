package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/73ai/openbotkit/internal/browser"
)

const defaultJSInstURL = "https://x.com/i/js_inst?c_name=ui_metrics"

const (
	guestActivateURL = "https://api.x.com/1.1/guest/activate.json"
	onboardingURL    = "https://api.x.com/1.1/onboarding/task.json"
)

type LoginResult struct {
	Session  *Session
	NeedsTFA bool
}

type loginFlow struct {
	httpClient *http.Client
	guestToken string
	flowToken  string
	jsInstURL  string // overridable for testing; defaults to defaultJSInstURL
}

func Login(username, password string) (*LoginResult, error) {
	httpClient := browser.NewClient()

	flow := &loginFlow{httpClient: httpClient}

	if err := flow.activateGuest(); err != nil {
		return nil, fmt.Errorf("activate guest: %w", err)
	}

	if err := flow.initLoginFlow(); err != nil {
		return nil, fmt.Errorf("init login flow: %w", err)
	}

	if err := flow.submitInstrumentation(); err != nil {
		return nil, fmt.Errorf("submit instrumentation: %w", err)
	}

	taskID, err := flow.submitUsername(username)
	if err != nil {
		return nil, fmt.Errorf("submit username: %w", err)
	}

	if taskID == "LoginEnterAlternateIdentifierSubtask" {
		return nil, fmt.Errorf("X requires additional verification (email/phone). Please log in via browser and use 'obk x auth login --token <token>' instead")
	}

	taskID, err = flow.submitPassword(password)
	if err != nil {
		return nil, fmt.Errorf("submit password: %w", err)
	}

	if taskID == "LoginTwoFactorAuthChallenge" {
		return &LoginResult{NeedsTFA: true}, nil
	}

	if taskID == "DenyLoginSubtask" {
		return nil, fmt.Errorf("login denied by X")
	}

	if err := flow.submitDuplicationCheck(); err != nil {
		return nil, fmt.Errorf("duplication check: %w", err)
	}

	session, err := flow.extractSession()
	if err != nil {
		return nil, err
	}

	return &LoginResult{Session: session}, nil
}

func LoginWithTFA(username, password, code string) (*LoginResult, error) {
	httpClient := browser.NewClient()

	flow := &loginFlow{httpClient: httpClient}

	if err := flow.activateGuest(); err != nil {
		return nil, fmt.Errorf("activate guest: %w", err)
	}

	if err := flow.initLoginFlow(); err != nil {
		return nil, fmt.Errorf("init login flow: %w", err)
	}

	if err := flow.submitInstrumentation(); err != nil {
		return nil, fmt.Errorf("submit instrumentation: %w", err)
	}

	if _, err := flow.submitUsername(username); err != nil {
		return nil, fmt.Errorf("submit username: %w", err)
	}

	taskID, err := flow.submitPassword(password)
	if err != nil {
		return nil, fmt.Errorf("submit password: %w", err)
	}

	if taskID != "LoginTwoFactorAuthChallenge" {
		return nil, fmt.Errorf("expected 2FA challenge but got: %s", taskID)
	}

	if err := flow.submitTFACode(code); err != nil {
		return nil, fmt.Errorf("submit 2FA code: %w", err)
	}

	if err := flow.submitDuplicationCheck(); err != nil {
		return nil, fmt.Errorf("duplication check: %w", err)
	}

	session, err := flow.extractSession()
	if err != nil {
		return nil, err
	}

	return &LoginResult{Session: session}, nil
}

func (f *loginFlow) activateGuest() error {
	req, err := http.NewRequest("POST", guestActivateURL, strings.NewReader("{}"))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		GuestToken string `json:"guest_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode guest token: %w", err)
	}
	if result.GuestToken == "" {
		return fmt.Errorf("empty guest token")
	}

	f.guestToken = result.GuestToken
	return nil
}

func (f *loginFlow) initLoginFlow() error {
	body := map[string]any{
		"input_flow_data": map[string]any{
			"flow_context": map[string]any{
				"debug_overrides": map[string]any{},
				"start_location": map[string]any{"location": "splash_screen"},
			},
		},
		"subtask_versions": map[string]any{},
	}

	resp, err := f.postTask(body)
	if err != nil {
		return err
	}

	f.flowToken = resp.FlowToken
	return nil
}

func (f *loginFlow) submitInstrumentation() error {
	instrResponse, err := f.solveInstrumentation()
	if err != nil {
		return fmt.Errorf("solve JS instrumentation: %w", err)
	}

	resp, err := f.postSubtask([]any{
		map[string]any{
			"subtask_id": "LoginJsInstrumentationSubtask",
			"js_instrumentation": map[string]any{
				"response": instrResponse,
				"link":     "next_link",
			},
		},
	})
	if err != nil {
		return err
	}
	f.flowToken = resp.FlowToken
	return nil
}

func (f *loginFlow) solveInstrumentation() (string, error) {
	fetchURL := f.jsInstURL
	if fetchURL == "" {
		fetchURL = defaultJSInstURL
	}
	req, err := http.NewRequest("GET", fetchURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch JS instrumentation: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read JS instrumentation: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("JS instrumentation fetch failed (status %d)", resp.StatusCode)
	}

	return SolveUIMetrics(string(body))
}

func (f *loginFlow) submitUsername(username string) (string, error) {
	resp, err := f.postSubtask([]any{
		map[string]any{
			"subtask_id": "LoginEnterUserIdentifierSSO",
			"settings_list": map[string]any{
				"setting_responses": []any{
					map[string]any{
						"key": "user_identifier",
						"response_data": map[string]any{
							"text_data": map[string]any{"result": username},
						},
					},
				},
				"link": "next_link",
			},
		},
	})
	if err != nil {
		return "", err
	}
	f.flowToken = resp.FlowToken
	return resp.TaskID(), nil
}

func (f *loginFlow) submitPassword(password string) (string, error) {
	resp, err := f.postSubtask([]any{
		map[string]any{
			"subtask_id": "LoginEnterPassword",
			"enter_password": map[string]any{
				"password": password,
				"link":     "next_link",
			},
		},
	})
	if err != nil {
		return "", err
	}
	f.flowToken = resp.FlowToken
	return resp.TaskID(), nil
}

func (f *loginFlow) submitTFACode(code string) error {
	resp, err := f.postSubtask([]any{
		map[string]any{
			"subtask_id": "LoginTwoFactorAuthChallenge",
			"enter_text": map[string]any{
				"text": code,
				"link": "next_link",
			},
		},
	})
	if err != nil {
		return err
	}
	f.flowToken = resp.FlowToken
	return nil
}

func (f *loginFlow) submitDuplicationCheck() error {
	resp, err := f.postSubtask([]any{
		map[string]any{
			"subtask_id": "AccountDuplicationCheck",
			"check_logged_in_account": map[string]any{
				"link": "AccountDuplicationCheck_false",
			},
		},
	})
	if err != nil {
		return err
	}
	f.flowToken = resp.FlowToken
	return nil
}

func (f *loginFlow) extractSession() (*Session, error) {
	jar := f.httpClient.Jar
	if jar == nil {
		return nil, fmt.Errorf("no cookie jar")
	}

	var authToken, ct0 string
	for _, domain := range []string{"https://x.com", "https://api.x.com", "https://twitter.com"} {
		cookies := jar.Cookies(mustParseURL(domain))
		for _, c := range cookies {
			switch c.Name {
			case "auth_token":
				authToken = c.Value
			case "ct0":
				ct0 = c.Value
			}
		}
	}

	if authToken == "" {
		return nil, fmt.Errorf("login succeeded but no auth_token cookie found")
	}

	session := &Session{AuthToken: authToken}
	if ct0 != "" {
		session.CSRFToken = ct0
	} else {
		session.CSRFToken = generateCSRFToken()
	}

	return session, nil
}

type taskResponse struct {
	FlowToken string `json:"flow_token"`
	Subtasks  []struct {
		SubtaskID string `json:"subtask_id"`
	} `json:"subtasks"`
}

func (r *taskResponse) TaskID() string {
	if len(r.Subtasks) > 0 {
		return r.Subtasks[0].SubtaskID
	}
	return ""
}

func (f *loginFlow) postTask(body any) (*taskResponse, error) {
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", onboardingURL+"?flow_name=login", strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, err
	}
	f.setHeaders(req)

	return f.doTask(req)
}

func (f *loginFlow) postSubtask(subtaskInputs []any) (*taskResponse, error) {
	body := map[string]any{
		"flow_token":     f.flowToken,
		"subtask_inputs": subtaskInputs,
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", onboardingURL, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, err
	}
	f.setHeaders(req)

	return f.doTask(req)
}

func (f *loginFlow) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("X-Guest-Token", f.guestToken)
	req.Header.Set("X-Twitter-Active-User", "yes")
	req.Header.Set("X-Twitter-Client-Language", "en")

	if jar := f.httpClient.Jar; jar != nil {
		for _, domain := range []string{"https://x.com", "https://api.x.com"} {
			for _, c := range jar.Cookies(mustParseURL(domain)) {
				if c.Name == "ct0" {
					req.Header.Set("X-Csrf-Token", c.Value)
				}
			}
		}
	}
}

func (f *loginFlow) doTask(req *http.Request) (*taskResponse, error) {
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("X API error (status %d): %s", resp.StatusCode, truncate(string(body), 200))
	}

	var result taskResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.TaskID() == "DenyLoginSubtask" {
		var raw map[string]any
		json.Unmarshal(body, &raw)
		errMsg := extractErrorMessage(raw)
		if errMsg != "" {
			return nil, fmt.Errorf("login denied: %s", errMsg)
		}
		return nil, fmt.Errorf("login denied by X")
	}

	return &result, nil
}

func extractErrorMessage(raw map[string]any) string {
	subtasks, ok := raw["subtasks"].([]any)
	if !ok {
		return ""
	}
	for _, st := range subtasks {
		stMap, ok := st.(map[string]any)
		if !ok {
			continue
		}
		if text, ok := stMap["secondary_text"].(map[string]any); ok {
			if t, ok := text["text"].(string); ok {
				return t
			}
		}
	}
	return ""
}

func mustParseURL(raw string) *url.URL {
	u, _ := url.Parse(raw)
	return u
}
