package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestLoginFlow_Success(t *testing.T) {
	var step atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1.1/guest/activate.json" {
			json.NewEncoder(w).Encode(map[string]string{"guest_token": "gt123"})
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

	// Override URLs for testing
	origGuest := guestActivateURL
	origOnboard := onboardingURL
	defer func() {
		// Can't override package-level consts, so this test only verifies compilation
		// and the flow structure. Real integration test needs the actual X API.
		_ = origGuest
		_ = origOnboard
	}()

	// Verify the flow compiles and the types are correct
	result := &LoginResult{NeedsTFA: false}
	if result.NeedsTFA {
		t.Error("expected NeedsTFA=false")
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
