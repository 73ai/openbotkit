package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync/atomic"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/priyanshujain/openbotkit/provider"
)

var fallbackMessages = map[string]string{
	"web_search":    "let me look that up",
	"web_fetch":     "one sec, checking that",
	"delegate_task": "give me a few minutes, looking into this",
	"subagent":      "lemme dig into this a bit",
	"gws_execute":   "one sec, let me check",
}

const defaultFallbackMsg = "one sec"

// slowTools are tools that take long enough to warrant an acknowledgment.
var slowTools = map[string]bool{
	"web_search":    true,
	"web_fetch":     true,
	"delegate_task": true,
	"subagent":      true,
	"gws_execute":   true,
}

type ackDecision struct {
	Action string `json:"action"`
	Emoji  string `json:"emoji,omitempty"`
	Text   string `json:"text,omitempty"`
}

type feedbackTimings struct {
	typingInterval time.Duration
	ackDelayMin    time.Duration
	ackDelayMax    time.Duration
}

var defaultTimings = feedbackTimings{
	typingInterval: 4 * time.Second,
	ackDelayMin:    8 * time.Second,
	ackDelayMax:    12 * time.Second,
}

type processingFeedback struct {
	bot       botSender
	chatID    int64
	messageID int
	userText  string
	provider  provider.Provider
	model     string
	timings   feedbackTimings

	cancel   context.CancelFunc
	done     chan struct{}
	signalCh chan string
	acked    atomic.Bool
}

func newProcessingFeedback(bot botSender, chatID int64, messageID int, userText string, p provider.Provider, model string) *processingFeedback {
	return &processingFeedback{
		bot:       bot,
		chatID:    chatID,
		messageID: messageID,
		userText:  userText,
		provider:  p,
		model:     model,
		timings:   defaultTimings,
		done:      make(chan struct{}),
		signalCh:  make(chan string, 1),
	}
}

func (f *processingFeedback) Start(ctx context.Context) {
	ctx, f.cancel = context.WithCancel(ctx)

	f.sendTyping()

	go f.loop(ctx)
}

func (f *processingFeedback) Signal(toolName string) {
	if !slowTools[toolName] {
		return
	}
	select {
	case f.signalCh <- toolName:
	default:
	}
}

func (f *processingFeedback) Stop() {
	if f.cancel != nil {
		f.cancel()
	}
	<-f.done
}

func (f *processingFeedback) loop(ctx context.Context) {
	defer close(f.done)

	typingTicker := time.NewTicker(f.timings.typingInterval)
	defer typingTicker.Stop()

	ackDelay := f.timings.ackDelayMin + time.Duration(rand.Int64N(int64(f.timings.ackDelayMax-f.timings.ackDelayMin)))
	ackTimer := time.NewTimer(ackDelay)
	defer ackTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-typingTicker.C:
			f.sendTyping()
		case toolName := <-f.signalCh:
			if f.acked.Load() {
				continue
			}
			f.acked.Store(true)
			ackTimer.Stop()
			go f.sendToolAck(ctx, toolName)
		case <-ackTimer.C:
			if f.acked.Load() {
				continue
			}
			f.acked.Store(true)
			f.sendText(defaultFallbackMsg)
		}
	}
}

func (f *processingFeedback) sendTyping() {
	action := tgbotapi.NewChatAction(f.chatID, tgbotapi.ChatTyping)
	if _, err := f.bot.Request(action); err != nil {
		slog.Debug("feedback: typing action failed", "error", err)
	}
}

func (f *processingFeedback) sendText(text string) {
	msg := tgbotapi.NewMessage(f.chatID, text)
	if _, err := f.bot.Send(msg); err != nil {
		slog.Debug("feedback: send text failed", "error", err)
	}
}

func (f *processingFeedback) sendReaction(emoji string) {
	reaction := []map[string]string{{"type": "emoji", "emoji": emoji}}
	reactionJSON, err := json.Marshal(reaction)
	if err != nil {
		slog.Debug("feedback: marshal reaction failed", "error", err)
		return
	}
	params := tgbotapi.Params{
		"chat_id":    fmt.Sprintf("%d", f.chatID),
		"message_id": fmt.Sprintf("%d", f.messageID),
		"reaction":   string(reactionJSON),
	}
	if _, err := f.bot.MakeRequest("setMessageReaction", params); err != nil {
		slog.Debug("feedback: reaction failed", "error", err)
	}
}

func (f *processingFeedback) sendToolAck(ctx context.Context, toolName string) {
	decision := f.decideAck(ctx, toolName)

	switch decision.Action {
	case "text":
		if decision.Text != "" {
			f.sendText(decision.Text)
		}
	case "reaction":
		if decision.Emoji != "" {
			f.sendReaction(decision.Emoji)
		}
	case "both":
		if decision.Emoji != "" {
			f.sendReaction(decision.Emoji)
		}
		if decision.Text != "" {
			f.sendText(decision.Text)
		}
	case "none":
		// just keep typing
	}
}

const ackModelTimeout = 3 * time.Second

func (f *processingFeedback) decideAck(ctx context.Context, toolName string) ackDecision {
	if f.provider == nil {
		return f.fallbackDecision(toolName)
	}

	ctx, cancel := context.WithTimeout(ctx, ackModelTimeout)
	defer cancel()

	resp, err := f.provider.Chat(ctx, provider.ChatRequest{
		Model:  f.model,
		System: ackSystemPrompt,
		Messages: []provider.Message{
			provider.NewTextMessage(provider.RoleUser,
				fmt.Sprintf("User message: %q\nTool running: %s", f.userText, toolName)),
		},
		MaxTokens: 100,
	})
	if err != nil {
		slog.Debug("feedback: model ack failed, using fallback", "error", err)
		return f.fallbackDecision(toolName)
	}

	var d ackDecision
	if err := json.Unmarshal([]byte(resp.TextContent()), &d); err != nil {
		slog.Debug("feedback: parse ack decision failed", "error", err)
		return f.fallbackDecision(toolName)
	}

	if d.Action == "" {
		return f.fallbackDecision(toolName)
	}
	return d
}

func (f *processingFeedback) fallbackDecision(toolName string) ackDecision {
	text, ok := fallbackMessages[toolName]
	if !ok {
		text = defaultFallbackMsg
	}
	return ackDecision{Action: "text", Text: text}
}

const ackSystemPrompt = `You decide how a personal assistant should acknowledge a message while processing it.
Given the user's message and what tool is running, respond with ONLY a JSON object (no markdown):
{"action":"text","text":"..."} — send a short text acknowledgment
{"action":"reaction","emoji":"..."} — react to the user's message with an emoji
{"action":"both","emoji":"...","text":"..."} — react AND send text
{"action":"none"} — stay silent, just keep typing

Rules:
- Be natural and varied, like a real person texting
- Keep text short and casual (e.g. "let me check", "one sec", "lemme look that up")
- Available emojis for reactions: 🤔 👀 👍 ⚡
- Don't always pick the same action — mix it up
- Match the tone of the user's message`
