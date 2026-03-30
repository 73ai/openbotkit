package client

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestLoginFlow_FullFlow(t *testing.T) {
	var step atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/guest" {
			json.NewEncoder(w).Encode(map[string]string{"guest_token": "gt123"})
			return
		}
		if r.URL.Path == "/js_inst" {
			w.Write([]byte(`function solve() {return {"rf": {"a": 1}, "s": "sig"}}`))
			return
		}

		s := step.Add(1)
		w.Header().Set("Content-Type", "application/json")

		switch s {
		case 1: // init flow
			json.NewEncoder(w).Encode(map[string]any{
				"flow_token": "ft1",
				"subtasks":   []map[string]any{{"subtask_id": "LoginJsInstrumentationSubtask"}},
			})
		case 2: // instrumentation
			json.NewEncoder(w).Encode(map[string]any{
				"flow_token": "ft2",
				"subtasks":   []map[string]any{{"subtask_id": "LoginEnterUserIdentifierSSO"}},
			})
		case 3: // username
			json.NewEncoder(w).Encode(map[string]any{
				"flow_token": "ft3",
				"subtasks":   []map[string]any{{"subtask_id": "LoginEnterPassword"}},
			})
		case 4: // password
			http.SetCookie(w, &http.Cookie{Name: "auth_token", Value: "at-test-value-long-enough"})
			http.SetCookie(w, &http.Cookie{Name: "ct0", Value: "csrf-test"})
			json.NewEncoder(w).Encode(map[string]any{
				"flow_token": "ft4",
				"subtasks":   []map[string]any{{"subtask_id": "AccountDuplicationCheck"}},
			})
		case 5: // duplication check
			json.NewEncoder(w).Encode(map[string]any{
				"flow_token": "ft5",
				"subtasks":   []map[string]any{{"subtask_id": "LoginSuccessSubtask"}},
			})
		}
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	httpClient := srv.Client()
	httpClient.Jar = jar

	flow := &loginFlow{
		httpClient: httpClient,
		guestURL:   srv.URL + "/guest",
		onboardURL: srv.URL + "/onboard",
		jsInstURL:  srv.URL + "/js_inst",
	}

	if err := flow.activateGuest(); err != nil {
		t.Fatalf("activateGuest: %v", err)
	}
	if flow.guestToken != "gt123" {
		t.Errorf("guestToken = %q, want gt123", flow.guestToken)
	}

	if err := flow.initLoginFlow(); err != nil {
		t.Fatalf("initLoginFlow: %v", err)
	}

	if err := flow.submitInstrumentation(); err != nil {
		t.Fatalf("submitInstrumentation: %v", err)
	}

	taskID, err := flow.submitUsername("testuser")
	if err != nil {
		t.Fatalf("submitUsername: %v", err)
	}
	if taskID != "LoginEnterPassword" {
		t.Errorf("taskID after username = %q, want LoginEnterPassword", taskID)
	}

	taskID, err = flow.submitPassword("testpass")
	if err != nil {
		t.Fatalf("submitPassword: %v", err)
	}
	if taskID != "AccountDuplicationCheck" {
		t.Errorf("taskID after password = %q, want AccountDuplicationCheck", taskID)
	}

	if err := flow.submitDuplicationCheck(); err != nil {
		t.Fatalf("submitDuplicationCheck: %v", err)
	}

	if flow.flowToken != "ft5" {
		t.Errorf("final flowToken = %q, want ft5", flow.flowToken)
	}
}

func TestExtractSession_FromCookies(t *testing.T) {
	jar, _ := cookiejar.New(nil)
	u := mustParseURL("https://x.com")
	jar.SetCookies(u, []*http.Cookie{
		{Name: "auth_token", Value: "at-test-value-long-enough"},
		{Name: "ct0", Value: "csrf-test"},
	})

	flow := &loginFlow{httpClient: &http.Client{Jar: jar}}
	session, err := flow.extractSession()
	if err != nil {
		t.Fatalf("extractSession: %v", err)
	}
	if session.AuthToken != "at-test-value-long-enough" {
		t.Errorf("AuthToken = %q, want at-test-value-long-enough", session.AuthToken)
	}
	if session.CSRFToken != "csrf-test" {
		t.Errorf("CSRFToken = %q, want csrf-test", session.CSRFToken)
	}
}

func TestExtractSession_NoCookieJar(t *testing.T) {
	flow := &loginFlow{httpClient: &http.Client{}}
	_, err := flow.extractSession()
	if err == nil {
		t.Error("expected error with no cookie jar")
	}
}

func TestExtractSession_NoAuthToken(t *testing.T) {
	jar, _ := cookiejar.New(nil)
	flow := &loginFlow{httpClient: &http.Client{Jar: jar}}
	_, err := flow.extractSession()
	if err == nil {
		t.Error("expected error with no auth_token cookie")
	}
}

func TestLoginResult_Types(t *testing.T) {
	r := &LoginResult{
		Session:  &Session{AuthToken: "test", CSRFToken: "csrf"},
		NeedsTFA: false,
	}
	if r.Session.AuthToken != "test" {
		t.Errorf("AuthToken = %q, want test", r.Session.AuthToken)
	}
	if r.NeedsTFA {
		t.Error("expected NeedsTFA=false")
	}
}

func TestTaskResponse_TaskID(t *testing.T) {
	r := &taskResponse{
		FlowToken: "ft1",
		Subtasks: []struct {
			SubtaskID string `json:"subtask_id"`
		}{
			{SubtaskID: "LoginEnterPassword"},
		},
	}
	if r.TaskID() != "LoginEnterPassword" {
		t.Errorf("TaskID() = %q, want LoginEnterPassword", r.TaskID())
	}
}

func TestTaskResponse_TaskID_Empty(t *testing.T) {
	r := &taskResponse{}
	if r.TaskID() != "" {
		t.Errorf("TaskID() = %q, want empty", r.TaskID())
	}
}

func TestExtractErrorMessage(t *testing.T) {
	raw := map[string]any{
		"subtasks": []any{
			map[string]any{
				"subtask_id": "DenyLoginSubtask",
				"secondary_text": map[string]any{
					"text": "Wrong password",
				},
			},
		},
	}
	msg := extractErrorMessage(raw)
	if msg != "Wrong password" {
		t.Errorf("extractErrorMessage = %q, want 'Wrong password'", msg)
	}
}

func TestExtractErrorMessage_NoSubtasks(t *testing.T) {
	msg := extractErrorMessage(map[string]any{})
	if msg != "" {
		t.Errorf("expected empty, got %q", msg)
	}
}

func TestSolveInstrumentation_MockServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Write([]byte(`function solve() {return {"rf": {"a": 1}, "s": "sig"}}`))
	}))
	defer srv.Close()

	flow := &loginFlow{
		httpClient: srv.Client(),
		jsInstURL:  srv.URL,
	}

	result, err := flow.solveInstrumentation()
	if err != nil {
		t.Fatalf("solveInstrumentation() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if parsed["s"] != "sig" {
		t.Errorf("result s = %v, want sig", parsed["s"])
	}
}

func TestSolveInstrumentation_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	flow := &loginFlow{
		httpClient: srv.Client(),
		jsInstURL:  srv.URL,
	}

	_, err := flow.solveInstrumentation()
	if err == nil {
		t.Error("expected error for server error response")
	}
}

func TestSolveInstrumentation_InvalidJS(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not valid js"))
	}))
	defer srv.Close()

	flow := &loginFlow{
		httpClient: srv.Client(),
		jsInstURL:  srv.URL,
	}

	_, err := flow.solveInstrumentation()
	if err == nil {
		t.Error("expected error for invalid JS")
	}
}
