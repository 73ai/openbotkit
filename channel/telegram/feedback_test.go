package telegram

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/priyanshujain/openbotkit/provider"
)

// feedbackBot tracks Send, Request, and MakeRequest calls separately.
type feedbackBot struct {
	mu           sync.Mutex
	sends        []tgbotapi.Chattable
	requestCalls []tgbotapi.Chattable
	makeRequests []makeRequestCall
}

type makeRequestCall struct {
	endpoint string
	params   tgbotapi.Params
}

func (b *feedbackBot) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sends = append(b.sends, c)
	return tgbotapi.Message{MessageID: 1}, nil
}

func (b *feedbackBot) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.requestCalls = append(b.requestCalls, c)
	return &tgbotapi.APIResponse{Ok: true}, nil
}

func (b *feedbackBot) MakeRequest(endpoint string, params tgbotapi.Params) (*tgbotapi.APIResponse, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.makeRequests = append(b.makeRequests, makeRequestCall{endpoint, params})
	return &tgbotapi.APIResponse{Ok: true}, nil
}

func (b *feedbackBot) sendCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.sends)
}

func (b *feedbackBot) requestCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.requestCalls)
}

func (b *feedbackBot) makeRequestCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.makeRequests)
}

// fastTimings returns short intervals for tests.
func fastTimings() feedbackTimings {
	return feedbackTimings{
		typingInterval: 100 * time.Millisecond,
		ackDelayMin:    500 * time.Millisecond,
		ackDelayMax:    600 * time.Millisecond,
	}
}

func newTestFeedback(bot botSender, p provider.Provider, model string) *processingFeedback {
	fb := newProcessingFeedback(bot, 123, 42, "test message", p, model)
	fb.timings = fastTimings()
	return fb
}

// feedbackProvider returns scripted responses for ack decisions.
type feedbackProvider struct {
	mu        sync.Mutex
	responses []*provider.ChatResponse
	idx       int
	err       error
}

func (p *feedbackProvider) Chat(_ context.Context, _ provider.ChatRequest) (*provider.ChatResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return nil, p.err
	}
	if p.idx >= len(p.responses) {
		return nil, fmt.Errorf("no more responses")
	}
	resp := p.responses[p.idx]
	p.idx++
	return resp, nil
}

func (p *feedbackProvider) StreamChat(_ context.Context, _ provider.ChatRequest) (<-chan provider.StreamEvent, error) {
	return nil, fmt.Errorf("not implemented")
}

func ackResponse(jsonStr string) *provider.ChatResponse {
	return &provider.ChatResponse{
		Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: jsonStr}},
		StopReason: provider.StopEndTurn,
	}
}

// --- Core behavior tests ---

func TestFeedback_TypingActionSentImmediately(t *testing.T) {
	bot := &feedbackBot{}
	fb := newTestFeedback(bot, nil, "")

	fb.Start(context.Background())
	time.Sleep(50 * time.Millisecond)
	fb.Stop()

	if bot.requestCount() < 1 {
		t.Fatal("expected at least 1 typing action via Request()")
	}
}

func TestFeedback_NoAckOnQuickStop(t *testing.T) {
	bot := &feedbackBot{}
	fb := newTestFeedback(bot, nil, "")

	fb.Start(context.Background())
	time.Sleep(50 * time.Millisecond)
	fb.Stop()

	if bot.sendCount() != 0 {
		t.Errorf("expected 0 text messages, got %d", bot.sendCount())
	}
}

func TestFeedback_TypingActionRepeats(t *testing.T) {
	bot := &feedbackBot{}
	fb := newTestFeedback(bot, nil, "")
	// Don't let ack fire — set high delay
	fb.timings.ackDelayMin = 10 * time.Second
	fb.timings.ackDelayMax = 11 * time.Second

	fb.Start(context.Background())
	time.Sleep(350 * time.Millisecond) // Should get ~3 typing actions at 100ms intervals + initial
	fb.Stop()

	if bot.requestCount() < 3 {
		t.Errorf("expected at least 3 typing actions, got %d", bot.requestCount())
	}
}

func TestFeedback_StopCancelsCleanly(t *testing.T) {
	bot := &feedbackBot{}
	fb := newTestFeedback(bot, nil, "")

	fb.Start(context.Background())
	fb.Stop()

	// done channel should be closed
	select {
	case <-fb.done:
	case <-time.After(time.Second):
		t.Fatal("done channel not closed after Stop()")
	}

	// No further sends after stop
	countBefore := bot.requestCount()
	time.Sleep(200 * time.Millisecond)
	if bot.requestCount() != countBefore {
		t.Error("typing actions continued after Stop()")
	}
}

// --- Tool signal tests ---

func TestFeedback_ToolSignalTriggersAck(t *testing.T) {
	bot := &feedbackBot{}
	fb := newTestFeedback(bot, nil, "") // nil provider → fallback
	fb.timings.ackDelayMin = 10 * time.Second
	fb.timings.ackDelayMax = 11 * time.Second

	fb.Start(context.Background())
	fb.Signal("web_search")
	time.Sleep(100 * time.Millisecond)
	fb.Stop()

	if bot.sendCount() != 1 {
		t.Fatalf("expected 1 ack message, got %d", bot.sendCount())
	}

	bot.mu.Lock()
	msg, ok := bot.sends[0].(tgbotapi.MessageConfig)
	bot.mu.Unlock()
	if !ok {
		t.Fatalf("expected MessageConfig, got %T", bot.sends[0])
	}
	if msg.Text != "let me look that up" {
		t.Errorf("ack text = %q, want %q", msg.Text, "let me look that up")
	}
}

func TestFeedback_OnlyOneAck(t *testing.T) {
	bot := &feedbackBot{}
	fb := newTestFeedback(bot, nil, "")
	fb.timings.ackDelayMin = 10 * time.Second
	fb.timings.ackDelayMax = 11 * time.Second

	fb.Start(context.Background())
	fb.Signal("web_search")
	time.Sleep(100 * time.Millisecond)
	fb.Signal("delegate_task")
	time.Sleep(100 * time.Millisecond)
	fb.Stop()

	if bot.sendCount() != 1 {
		t.Errorf("expected exactly 1 ack message, got %d", bot.sendCount())
	}
}

func TestFeedback_SignalAfterAckIgnored(t *testing.T) {
	bot := &feedbackBot{}
	fb := newTestFeedback(bot, nil, "")

	fb.Start(context.Background())
	// Wait for timeout ack
	time.Sleep(700 * time.Millisecond)

	sendsBefore := bot.sendCount()
	fb.Signal("web_search")
	time.Sleep(100 * time.Millisecond)
	fb.Stop()

	if bot.sendCount() != sendsBefore {
		t.Errorf("signal after ack should be ignored; sends before=%d, after=%d",
			sendsBefore, bot.sendCount())
	}
}

// --- Timeout fallback tests ---

func TestFeedback_TimeoutAckAfterDelay(t *testing.T) {
	bot := &feedbackBot{}
	fb := newTestFeedback(bot, nil, "")

	fb.Start(context.Background())
	time.Sleep(800 * time.Millisecond) // past ackDelayMax (600ms)
	fb.Stop()

	if bot.sendCount() != 1 {
		t.Fatalf("expected 1 timeout ack, got %d", bot.sendCount())
	}

	bot.mu.Lock()
	msg, ok := bot.sends[0].(tgbotapi.MessageConfig)
	bot.mu.Unlock()
	if !ok {
		t.Fatalf("expected MessageConfig, got %T", bot.sends[0])
	}
	if msg.Text != defaultFallbackMsg {
		t.Errorf("text = %q, want %q", msg.Text, defaultFallbackMsg)
	}
}

func TestFeedback_SignalBeforeTimeoutPreventsTimeout(t *testing.T) {
	bot := &feedbackBot{}
	fb := newTestFeedback(bot, nil, "")

	fb.Start(context.Background())
	fb.Signal("web_search")
	time.Sleep(800 * time.Millisecond) // past timeout
	fb.Stop()

	if bot.sendCount() != 1 {
		t.Errorf("expected exactly 1 ack (signal-driven), got %d", bot.sendCount())
	}
}

// --- Model-driven ack tests ---

func TestFeedback_ModelDecision_TextOnly(t *testing.T) {
	bot := &feedbackBot{}
	p := &feedbackProvider{
		responses: []*provider.ChatResponse{
			ackResponse(`{"action":"text","text":"checking..."}`),
		},
	}
	fb := newTestFeedback(bot, p, "fast-model")
	fb.timings.ackDelayMin = 10 * time.Second
	fb.timings.ackDelayMax = 11 * time.Second

	fb.Start(context.Background())
	fb.Signal("web_search")
	time.Sleep(200 * time.Millisecond)
	fb.Stop()

	if bot.sendCount() != 1 {
		t.Fatalf("expected 1 text message, got %d", bot.sendCount())
	}
	bot.mu.Lock()
	msg := bot.sends[0].(tgbotapi.MessageConfig)
	bot.mu.Unlock()
	if msg.Text != "checking..." {
		t.Errorf("text = %q, want %q", msg.Text, "checking...")
	}
	if bot.makeRequestCount() != 0 {
		t.Error("no reaction expected for text-only")
	}
}

func TestFeedback_ModelDecision_ReactionOnly(t *testing.T) {
	bot := &feedbackBot{}
	p := &feedbackProvider{
		responses: []*provider.ChatResponse{
			ackResponse(`{"action":"reaction","emoji":"🤔"}`),
		},
	}
	fb := newTestFeedback(bot, p, "fast-model")
	fb.timings.ackDelayMin = 10 * time.Second
	fb.timings.ackDelayMax = 11 * time.Second

	fb.Start(context.Background())
	fb.Signal("web_search")
	time.Sleep(200 * time.Millisecond)
	fb.Stop()

	if bot.sendCount() != 0 {
		t.Errorf("expected 0 text messages for reaction-only, got %d", bot.sendCount())
	}
	if bot.makeRequestCount() != 1 {
		t.Fatalf("expected 1 reaction, got %d", bot.makeRequestCount())
	}
	bot.mu.Lock()
	req := bot.makeRequests[0]
	bot.mu.Unlock()
	if req.endpoint != "setMessageReaction" {
		t.Errorf("endpoint = %q", req.endpoint)
	}
}

func TestFeedback_ModelDecision_Both(t *testing.T) {
	bot := &feedbackBot{}
	p := &feedbackProvider{
		responses: []*provider.ChatResponse{
			ackResponse(`{"action":"both","emoji":"👀","text":"one sec"}`),
		},
	}
	fb := newTestFeedback(bot, p, "fast-model")
	fb.timings.ackDelayMin = 10 * time.Second
	fb.timings.ackDelayMax = 11 * time.Second

	fb.Start(context.Background())
	fb.Signal("delegate_task")
	time.Sleep(200 * time.Millisecond)
	fb.Stop()

	if bot.sendCount() != 1 {
		t.Fatalf("expected 1 text for 'both', got %d", bot.sendCount())
	}
	if bot.makeRequestCount() != 1 {
		t.Fatalf("expected 1 reaction for 'both', got %d", bot.makeRequestCount())
	}
}

func TestFeedback_ModelDecision_None(t *testing.T) {
	bot := &feedbackBot{}
	p := &feedbackProvider{
		responses: []*provider.ChatResponse{
			ackResponse(`{"action":"none"}`),
		},
	}
	fb := newTestFeedback(bot, p, "fast-model")
	fb.timings.ackDelayMin = 10 * time.Second
	fb.timings.ackDelayMax = 11 * time.Second

	fb.Start(context.Background())
	fb.Signal("web_search")
	time.Sleep(200 * time.Millisecond)
	fb.Stop()

	if bot.sendCount() != 0 {
		t.Errorf("expected 0 text messages for 'none', got %d", bot.sendCount())
	}
	if bot.makeRequestCount() != 0 {
		t.Errorf("expected 0 reactions for 'none', got %d", bot.makeRequestCount())
	}
}

func TestFeedback_ModelFailure_Fallback(t *testing.T) {
	bot := &feedbackBot{}
	p := &feedbackProvider{err: fmt.Errorf("api error")}
	fb := newTestFeedback(bot, p, "fast-model")
	fb.timings.ackDelayMin = 10 * time.Second
	fb.timings.ackDelayMax = 11 * time.Second

	fb.Start(context.Background())
	fb.Signal("web_search")
	time.Sleep(200 * time.Millisecond)
	fb.Stop()

	if bot.sendCount() != 1 {
		t.Fatalf("expected 1 fallback message, got %d", bot.sendCount())
	}
	bot.mu.Lock()
	msg := bot.sends[0].(tgbotapi.MessageConfig)
	bot.mu.Unlock()
	if msg.Text != "let me look that up" {
		t.Errorf("fallback text = %q, want %q", msg.Text, "let me look that up")
	}
}

// --- Edge cases ---

func TestFeedback_NilProvider_UsesFallback(t *testing.T) {
	bot := &feedbackBot{}
	fb := newTestFeedback(bot, nil, "")
	fb.timings.ackDelayMin = 10 * time.Second
	fb.timings.ackDelayMax = 11 * time.Second

	fb.Start(context.Background())
	fb.Signal("delegate_task")
	time.Sleep(100 * time.Millisecond)
	fb.Stop()

	if bot.sendCount() != 1 {
		t.Fatalf("expected 1 fallback message, got %d", bot.sendCount())
	}
	bot.mu.Lock()
	msg := bot.sends[0].(tgbotapi.MessageConfig)
	bot.mu.Unlock()
	if msg.Text != "give me a few minutes, looking into this" {
		t.Errorf("fallback text = %q", msg.Text)
	}
}

func TestFeedback_ContextCancellation(t *testing.T) {
	bot := &feedbackBot{}
	fb := newTestFeedback(bot, nil, "")

	ctx, cancel := context.WithCancel(context.Background())
	fb.Start(ctx)

	cancel()
	// Stop should return quickly
	done := make(chan struct{})
	go func() {
		fb.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Stop() did not return after context cancellation")
	}
}

func TestFeedback_FastToolIgnored(t *testing.T) {
	bot := &feedbackBot{}
	fb := newTestFeedback(bot, nil, "")
	fb.timings.ackDelayMin = 10 * time.Second
	fb.timings.ackDelayMax = 11 * time.Second

	fb.Start(context.Background())
	fb.Signal("bash")
	fb.Signal("some_unknown_tool")
	time.Sleep(100 * time.Millisecond)
	fb.Stop()

	if bot.sendCount() != 0 {
		t.Errorf("fast/unknown tools should not trigger ack, got %d sends", bot.sendCount())
	}
}

func TestFeedback_ModelDecision_EmptyTextIgnored(t *testing.T) {
	bot := &feedbackBot{}
	p := &feedbackProvider{
		responses: []*provider.ChatResponse{
			ackResponse(`{"action":"text","text":""}`),
		},
	}
	fb := newTestFeedback(bot, p, "fast-model")
	fb.timings.ackDelayMin = 10 * time.Second
	fb.timings.ackDelayMax = 11 * time.Second

	fb.Start(context.Background())
	fb.Signal("web_search")
	time.Sleep(200 * time.Millisecond)
	fb.Stop()

	if bot.sendCount() != 0 {
		t.Errorf("empty text should not send a message, got %d", bot.sendCount())
	}
}

func TestFeedback_TypingContinuesDuringModelCall(t *testing.T) {
	bot := &feedbackBot{}
	// Provider that takes 300ms to respond
	p := &feedbackProvider{
		responses: []*provider.ChatResponse{
			ackResponse(`{"action":"text","text":"checking"}`),
		},
	}
	fb := newTestFeedback(bot, p, "fast-model")
	fb.timings.typingInterval = 50 * time.Millisecond
	fb.timings.ackDelayMin = 10 * time.Second
	fb.timings.ackDelayMax = 11 * time.Second

	fb.Start(context.Background())
	fb.Signal("web_search")
	time.Sleep(300 * time.Millisecond)
	fb.Stop()

	// Typing should have continued while model call ran
	if bot.requestCount() < 3 {
		t.Errorf("expected typing to continue during model call, got %d typing actions", bot.requestCount())
	}
}
