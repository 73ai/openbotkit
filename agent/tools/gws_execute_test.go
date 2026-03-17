package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/73ai/openbotkit/internal/skills"
	"github.com/73ai/openbotkit/oauth/google"
)

func setupGWSTest(t *testing.T, approveAll bool, scopes map[string]bool) (*GWSExecuteTool, *mockInteractor, *mockRunner) {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tokens.db")
	credPath := writeTestCreds(t, dir)

	store, err := google.NewTokenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	tok := &oauth2.Token{
		AccessToken:  "test-token",
		RefreshToken: "test-refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	if err := store.SaveToken("user@test.com", tok, []string{"openid", "email", "calendar"}); err != nil {
		t.Fatal(err)
	}
	store.Close()

	g := google.New(google.Config{
		CredentialsFile: credPath,
		TokenDBPath:     dbPath,
	})
	bridge := NewTokenBridge(g, "user@test.com")

	inter := &mockInteractor{approveAll: approveAll}
	runner := &mockRunner{outputs: make(map[string]string)}

	if scopes == nil {
		scopes = map[string]bool{"https://www.googleapis.com/auth/calendar": true}
	}
	checker := &mockScopeChecker{scopes: scopes}

	manifest := &skills.Manifest{
		Skills: map[string]skills.SkillEntry{
			"gws-calendar-list":   {Source: "gws", Scopes: []string{"calendar"}, Write: false},
			"gws-calendar-edit":   {Source: "gws", Scopes: []string{"calendar"}, Write: true},
			"gws-calendar-agenda": {Source: "gws", Scopes: []string{"calendar"}, Write: false},
			"gws-calendar-insert": {Source: "gws", Scopes: []string{"calendar"}, Write: true},
		},
	}

	tool := NewGWSExecuteTool(GWSToolConfig{
		Interactor:   inter,
		ScopeChecker: checker,
		Bridge:       bridge,
		ScopeWaiter:  google.NewScopeWaiter(),
		Google:       g,
		Account:      "user@test.com",
		Manifest:     manifest,
		Runner:       runner,
		AuthTimeout:  100 * time.Millisecond,
	})

	return tool, inter, runner
}

func TestGWSExecute_ReadPath(t *testing.T) {
	tool, inter, runner := setupGWSTest(t, false, nil)
	runner.outputs["calendar events list"] = `{"items":[]}`

	input, _ := json.Marshal(gwsInput{Command: "calendar events list"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != `{"items":[]}` {
		t.Errorf("result = %q", result)
	}
	// No approval should have been requested.
	if len(inter.approvals) > 0 {
		t.Error("read path should not request approval")
	}
	// Token should be in env.
	if len(runner.ran) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runner.ran))
	}
	envStr := strings.Join(runner.ran[0].env, " ")
	if !strings.Contains(envStr, "GOOGLE_WORKSPACE_CLI_TOKEN=") {
		t.Errorf("token not injected in env: %v", runner.ran[0].env)
	}
}

func TestGWSExecute_WriteApproved(t *testing.T) {
	tool, inter, runner := setupGWSTest(t, true, nil)
	runner.outputs["calendar +insert --json {}"] = "event created"

	input, _ := json.Marshal(gwsInput{Command: "calendar +insert --json {}"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "event created" {
		t.Errorf("result = %q", result)
	}
	if len(inter.approvals) != 1 {
		t.Errorf("expected 1 approval request, got %d", len(inter.approvals))
	}
	// Should have "Done." notification.
	hasNotification := false
	for _, n := range inter.notified {
		if n == "Done." {
			hasNotification = true
		}
	}
	if !hasNotification {
		t.Errorf("missing Done notification, got %v", inter.notified)
	}
}

func TestGWSExecute_WriteDenied(t *testing.T) {
	tool, inter, runner := setupGWSTest(t, false, nil)

	input, _ := json.Marshal(gwsInput{Command: "calendar +insert --json {}"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "denied_by_user" {
		t.Errorf("result = %q, want denied_by_user", result)
	}
	if len(runner.ran) > 0 {
		t.Error("command should not have been executed when denied")
	}
	hasNotification := false
	for _, n := range inter.notified {
		if n == "Action not performed." {
			hasNotification = true
		}
	}
	if !hasNotification {
		t.Errorf("missing denial notification, got %v", inter.notified)
	}
	_ = inter
}

func TestGWSExecute_MissingScopeTimeout(t *testing.T) {
	// Use scopes map that doesn't have calendar.
	tool, inter, _ := setupGWSTest(t, false, map[string]bool{})

	input, _ := json.Marshal(gwsInput{Command: "calendar events list"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected auth timeout error")
	}
	if !strings.Contains(err.Error(), "auth") {
		t.Errorf("error = %v", err)
	}
	// Should have sent auth link.
	if len(inter.links) == 0 {
		t.Error("expected auth link to be sent")
	}
}

func TestGWSExecute_MissingScopeSignaled(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tokens.db")
	credPath := writeTestCreds(t, dir)

	store, err := google.NewTokenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	tok := &oauth2.Token{
		AccessToken:  "test-token",
		RefreshToken: "test-refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	if err := store.SaveToken("user@test.com", tok, []string{"openid", "email"}); err != nil {
		t.Fatal(err)
	}
	store.Close()

	g := google.New(google.Config{
		CredentialsFile: credPath,
		TokenDBPath:     dbPath,
	})
	bridge := NewTokenBridge(g, "user@test.com")
	linkCh := make(chan struct{ text, url string }, 1)
	inter := &mockInteractor{approveAll: false, linkCh: linkCh}
	runner := &mockRunner{outputs: map[string]string{"calendar events list": `{"items":[]}`}}

	waiter := google.NewScopeWaiter()
	manifest := &skills.Manifest{
		Skills: map[string]skills.SkillEntry{
			"gws-calendar-list": {Source: "gws", Scopes: []string{"calendar"}},
		},
	}

	// Set up checker that starts without calendar, then has it after consent.
	checker := &toggleScopeChecker{hasAfterSignal: true}

	tool := NewGWSExecuteTool(GWSToolConfig{
		Interactor:   inter,
		ScopeChecker: checker,
		Bridge:       bridge,
		ScopeWaiter:  waiter,
		Google:       g,
		Account:      "user@test.com",
		Manifest:     manifest,
		Runner:       runner,
		AuthTimeout:  5 * time.Second,
	})

	// Run in background, signal the waiter after auth link is sent.
	done := make(chan struct{})
	var result string
	var execErr error
	go func() {
		input, _ := json.Marshal(gwsInput{Command: "calendar events list"})
		result, execErr = tool.Execute(context.Background(), input)
		close(done)
	}()

	// Wait for the auth link to be sent via channel (race-free).
	var link struct{ text, url string }
	select {
	case link = <-linkCh:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for auth link")
	}

	// Extract state from the auth link URL.
	linkURL := link.url
	stateIdx := strings.Index(linkURL, "state=")
	if stateIdx < 0 {
		t.Fatal("auth link missing state param")
	}
	state := linkURL[stateIdx+6:]
	if ampIdx := strings.Index(state, "&"); ampIdx >= 0 {
		state = state[:ampIdx]
	}

	waiter.Signal(state, nil)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for execute")
	}

	if execErr != nil {
		t.Fatalf("Execute: %v", execErr)
	}
	if result != `{"items":[]}` {
		t.Errorf("result = %q", result)
	}
}

// toggleScopeChecker returns false on first call, true after.
type toggleScopeChecker struct {
	called         int
	hasAfterSignal bool
}

func (c *toggleScopeChecker) HasScopes(_ string, _ []string) (bool, error) {
	c.called++
	if c.called == 1 {
		return false, nil
	}
	return c.hasAfterSignal, nil
}

func TestGWSExecute_ReadOnlyHelperNotWrite(t *testing.T) {
	tool, inter, runner := setupGWSTest(t, false, nil)
	runner.outputs["calendar +agenda"] = `{"events":[]}`

	input, _ := json.Marshal(gwsInput{Command: "calendar +agenda"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != `{"events":[]}` {
		t.Errorf("result = %q", result)
	}
	if len(inter.approvals) > 0 {
		t.Error("+agenda is read-only, should not trigger approval")
	}
}

func TestGWSExecute_KeywordNotWrite(t *testing.T) {
	tool, inter, runner := setupGWSTest(t, false, nil)
	runner.outputs["calendar events delete --id 123"] = "deleted"

	input, _ := json.Marshal(gwsInput{Command: "calendar events delete --id 123"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "deleted" {
		t.Errorf("result = %q", result)
	}
	if len(inter.approvals) > 0 {
		t.Error("keyword 'delete' without '+' prefix should not trigger approval")
	}
}

func TestGWSExecute_ScopesForService(t *testing.T) {
	manifest := &skills.Manifest{
		Skills: map[string]skills.SkillEntry{
			"gws-docs-read":  {Source: "gws", Scopes: []string{"docs"}},
			"gws-drive-list": {Source: "gws", Scopes: []string{"drive"}},
		},
	}
	tool := &GWSExecuteTool{manifest: manifest}

	tests := []struct {
		service string
		want    string
	}{
		{"docs", "https://www.googleapis.com/auth/documents"},
		{"drive", "https://www.googleapis.com/auth/drive"},
		// gmail has no manifest entry but exists in serviceToScope — fallback works.
		{"gmail", "https://www.googleapis.com/auth/gmail.modify"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		scopes := tool.scopesForService(tt.service)
		if tt.want == "" {
			if scopes != nil {
				t.Errorf("scopesForService(%q) = %v, want nil", tt.service, scopes)
			}
		} else if len(scopes) != 1 || scopes[0] != tt.want {
			t.Errorf("scopesForService(%q) = %v, want [%s]", tt.service, scopes, tt.want)
		}
	}
}

func TestGWSExecute_StripGWSPrefix(t *testing.T) {
	tool, _, runner := setupGWSTest(t, false, nil)
	runner.outputs["calendar events list"] = `{"items":[]}`

	input, _ := json.Marshal(gwsInput{Command: "gws calendar events list"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != `{"items":[]}` {
		t.Errorf("result = %q", result)
	}
	if len(runner.ran) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runner.ran))
	}
	if runner.ran[0].args[0] != "calendar" {
		t.Errorf("first arg = %q, want 'calendar'", runner.ran[0].args[0])
	}
}

func TestGWSExecute_StructuredParams(t *testing.T) {
	tool, _, runner := setupGWSTest(t, false, map[string]bool{
		"https://www.googleapis.com/auth/calendar": true,
		"https://www.googleapis.com/auth/drive":    true,
	})
	runner.outputs[`drive files list --params {"orderBy":"modifiedTime desc","pageSize":5,"q":"mimeType='application/vnd.google-apps.document'"}`] = `{"files":[]}`

	input, _ := json.Marshal(map[string]any{
		"command": "drive files list",
		"params": map[string]any{
			"q":       "mimeType='application/vnd.google-apps.document'",
			"orderBy": "modifiedTime desc",
			"pageSize": 5,
		},
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != `{"files":[]}` {
		t.Errorf("result = %q", result)
	}
	if len(runner.ran) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runner.ran))
	}
	// Verify --params is a single arg with valid JSON.
	args := runner.ran[0].args
	paramsIdx := -1
	for i, a := range args {
		if a == "--params" {
			paramsIdx = i
			break
		}
	}
	if paramsIdx < 0 || paramsIdx+1 >= len(args) {
		t.Fatal("--params flag not found in args")
	}
	if !json.Valid([]byte(args[paramsIdx+1])) {
		t.Errorf("--params value is not valid JSON: %s", args[paramsIdx+1])
	}
}

func TestGWSExecute_StructuredBody(t *testing.T) {
	tool, interactor, runner := setupGWSTest(t, false, nil)
	interactor.approveAll = true
	runner.outputs[`calendar +insert --json {"location":"Room 1","summary":"Team meeting"}`] = `{"id":"abc"}`

	input, _ := json.Marshal(map[string]any{
		"command": "calendar +insert",
		"body": map[string]any{
			"summary":  "Team meeting",
			"location": "Room 1",
		},
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != `{"id":"abc"}` {
		t.Errorf("result = %q", result)
	}
}

func TestGWSExecute_TokenExpiredTriggersReAuth(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tokens.db")
	credPath := writeTestCreds(t, dir)

	store, err := google.NewTokenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	// Save a token with scopes but expired and no refresh token.
	tok := &oauth2.Token{
		AccessToken: "expired-token",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(-time.Hour), // expired
	}
	if err := store.SaveToken("user@test.com", tok, []string{"openid", "email", "calendar"}); err != nil {
		t.Fatal(err)
	}
	store.Close()

	g := google.New(google.Config{
		CredentialsFile: credPath,
		TokenDBPath:     dbPath,
	})
	bridge := NewTokenBridge(g, "user@test.com")
	linkCh := make(chan struct{ text, url string }, 1)
	inter := &mockInteractor{approveAll: false, linkCh: linkCh}
	runner := &mockRunner{outputs: map[string]string{"calendar events list": `{"items":[]}`}}

	waiter := google.NewScopeWaiter()
	manifest := &skills.Manifest{
		Skills: map[string]skills.SkillEntry{
			"gws-calendar-list": {Source: "gws", Scopes: []string{"calendar"}},
		},
	}

	// Scopes ARE present — the token just can't be refreshed.
	checker := &mockScopeChecker{scopes: map[string]bool{
		"https://www.googleapis.com/auth/calendar": true,
	}}

	tool := NewGWSExecuteTool(GWSToolConfig{
		Interactor:   inter,
		ScopeChecker: checker,
		Bridge:       bridge,
		ScopeWaiter:  waiter,
		Google:       g,
		Account:      "user@test.com",
		Manifest:     manifest,
		Runner:       runner,
		AuthTimeout:  5 * time.Second,
	})

	done := make(chan struct{})
	var result string
	var execErr error
	go func() {
		input, _ := json.Marshal(gwsInput{Command: "calendar events list"})
		result, execErr = tool.Execute(context.Background(), input)
		close(done)
	}()

	// Wait for the re-auth link to be sent.
	var link struct{ text, url string }
	select {
	case link = <-linkCh:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for re-auth link")
	}

	// Simulate OAuth callback: save a fresh token, then signal the waiter.
	store2, err := google.NewTokenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	freshTok := &oauth2.Token{
		AccessToken:  "fresh-token",
		RefreshToken: "fresh-refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	if err := store2.SaveToken("user@test.com", freshTok, []string{"openid", "email", "calendar"}); err != nil {
		t.Fatal(err)
	}
	store2.Close()

	// Extract state and signal.
	linkURL := link.url
	state, _, _ := strings.Cut(linkURL[strings.Index(linkURL, "state=")+6:], "&")
	waiter.Signal(state, nil)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for execute")
	}

	if execErr != nil {
		t.Fatalf("Execute: %v", execErr)
	}
	if result != `{"items":[]}` {
		t.Errorf("result = %q", result)
	}
	// Verify an auth link was sent (re-auth triggered).
	if len(inter.links) == 0 {
		t.Error("expected re-auth link to be sent")
	}
}

func TestGWSExecute_EmptyCommand(t *testing.T) {
	tool, _, _ := setupGWSTest(t, false, nil)
	input, _ := json.Marshal(gwsInput{Command: ""})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for empty command")
	}
}

// writeTestCreds for gws_execute_test.go is defined in token_bridge_test.go
// since both are in the same package.

func TestGWSExecute_Name(t *testing.T) {
	dir := t.TempDir()
	credPath := writeTestCreds(t, dir)
	dbPath := filepath.Join(dir, "tokens.db")
	// Write an empty token db so NewTokenStore can init.
	os.WriteFile(dbPath, nil, 0600)

	g := google.New(google.Config{CredentialsFile: credPath, TokenDBPath: dbPath})
	tool := NewGWSExecuteTool(GWSToolConfig{
		Interactor:   &mockInteractor{},
		ScopeChecker: &mockScopeChecker{scopes: map[string]bool{}},
		Bridge:       NewTokenBridge(g, ""),
		ScopeWaiter:  google.NewScopeWaiter(),
		Google:       g,
		Manifest:     &skills.Manifest{Skills: map[string]skills.SkillEntry{}},
		Runner:       &mockRunner{outputs: map[string]string{}},
	})
	if tool.Name() != "gws_execute" {
		t.Errorf("Name() = %q", tool.Name())
	}
	if !json.Valid(tool.InputSchema()) {
		t.Error("invalid input schema")
	}
}

func TestGWSExecute_ServiceDisabledAnnotation(t *testing.T) {
	tool, inter, runner := setupGWSTest(t, false, nil)

	apiError := `{"error":{"code":403,"message":"Google Calendar API has not been used in project 123456 before or it is disabled. Enable it by visiting https://console.developers.google.com/apis/api/calendar-json.googleapis.com/overview?project=123456 then retry.","status":"PERMISSION_DENIED","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"SERVICE_DISABLED","domain":"googleapis.com"}]}}`
	runner.outputs["calendar events list"] = apiError
	runner.errs = map[string]error{"calendar events list": fmt.Errorf("exit status 1")}

	input, _ := json.Marshal(gwsInput{Command: "calendar events list"})
	result, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for disabled API")
	}
	if !strings.Contains(result, "SERVICE_DISABLED") {
		t.Error("result should contain original SERVICE_DISABLED error")
	}
	if !strings.Contains(result, "console.developers.google.com") {
		t.Error("result should preserve the activation URL")
	}
	if !strings.Contains(result, "The Google API for this service is not enabled") {
		t.Error("result should contain annotation note")
	}
	// No re-auth should be triggered (re-auth can't fix a disabled API).
	if len(inter.links) > 0 {
		t.Error("SERVICE_DISABLED should not trigger re-auth")
	}
	// Command should only run once (no retry).
	if len(runner.ran) != 1 {
		t.Errorf("expected 1 run, got %d", len(runner.ran))
	}
}

// scopeRetryRunner fails with ACCESS_TOKEN_SCOPE_INSUFFICIENT on first call,
// then succeeds on retry.
type scopeRetryRunner struct {
	calls int
	ran   []struct{ args, env []string }
}

func (r *scopeRetryRunner) Run(_ context.Context, args []string, env []string) (string, error) {
	r.ran = append(r.ran, struct{ args, env []string }{args, env})
	r.calls++
	if r.calls == 1 {
		return `{"error":{"code":403,"status":"PERMISSION_DENIED","details":[{"reason":"ACCESS_TOKEN_SCOPE_INSUFFICIENT"}]}}`, fmt.Errorf("exit status 1")
	}
	return `{"items":[]}`, nil
}

func TestGWSExecute_ScopeInsufficientRetry(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tokens.db")
	credPath := writeTestCreds(t, dir)

	store, err := google.NewTokenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	tok := &oauth2.Token{
		AccessToken:  "test-token",
		RefreshToken: "test-refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	if err := store.SaveToken("user@test.com", tok, []string{"openid", "email", "calendar"}); err != nil {
		t.Fatal(err)
	}
	store.Close()

	g := google.New(google.Config{
		CredentialsFile: credPath,
		TokenDBPath:     dbPath,
	})
	bridge := NewTokenBridge(g, "user@test.com")
	linkCh := make(chan struct{ text, url string }, 1)
	inter := &mockInteractor{approveAll: false, linkCh: linkCh}
	runner := &scopeRetryRunner{}

	waiter := google.NewScopeWaiter()
	checker := &mockScopeChecker{scopes: map[string]bool{
		"https://www.googleapis.com/auth/calendar": true,
	}}

	tool := NewGWSExecuteTool(GWSToolConfig{
		Interactor:   inter,
		ScopeChecker: checker,
		Bridge:       bridge,
		ScopeWaiter:  waiter,
		Google:       g,
		Account:      "user@test.com",
		Manifest: &skills.Manifest{
			Skills: map[string]skills.SkillEntry{
				"gws-calendar-list": {Source: "gws", Scopes: []string{"calendar"}},
			},
		},
		Runner:      runner,
		AuthTimeout: 5 * time.Second,
	})

	done := make(chan struct{})
	var result string
	var execErr error
	go func() {
		input, _ := json.Marshal(gwsInput{Command: "calendar events list"})
		result, execErr = tool.Execute(context.Background(), input)
		close(done)
	}()

	// Wait for the re-auth link triggered by scope insufficient.
	var link struct{ text, url string }
	select {
	case link = <-linkCh:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for re-auth link")
	}

	// Extract state and signal success.
	stateIdx := strings.Index(link.url, "state=")
	if stateIdx < 0 {
		t.Fatal("auth link missing state param")
	}
	state := link.url[stateIdx+6:]
	if ampIdx := strings.Index(state, "&"); ampIdx >= 0 {
		state = state[:ampIdx]
	}
	waiter.Signal(state, nil)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for execute")
	}

	if execErr != nil {
		t.Fatalf("Execute: %v", execErr)
	}
	if result != `{"items":[]}` {
		t.Errorf("result = %q", result)
	}
	if runner.calls != 2 {
		t.Errorf("expected 2 runner calls (fail + retry), got %d", runner.calls)
	}
}

func TestGWSExecute_TruncatesLargeOutput(t *testing.T) {
	tool, _, runner := setupGWSTest(t, false, nil)
	// Mock runner returns 60KB output.
	bigOutput := strings.Repeat("line\n", 12000)
	runner.outputs["calendar events list"] = bigOutput

	input, _ := json.Marshal(gwsInput{Command: "calendar events list"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result, "[truncated: showing 500+500 of") {
		t.Error("expected head+tail truncation marker")
	}
	if len(result) >= len(bigOutput) {
		t.Errorf("result should be truncated: got %d bytes, original %d", len(result), len(bigOutput))
	}
}
