package channel

import (
	"context"
	"errors"
	"testing"
)

type recordingPusher struct{ messages []string }

func (p *recordingPusher) Push(_ context.Context, message string) error {
	p.messages = append(p.messages, message)
	return nil
}

func TestLazyPusherRetriesCreateUntilItSucceeds(t *testing.T) {
	created := 0
	real := &recordingPusher{}
	lazy := NewLazyPusher(func() (Pusher, error) {
		created++
		if created == 1 {
			return nil, errors.New("create telegram bot: dial tcp: lookup api.telegram.org: no such host")
		}
		return real, nil
	})

	ctx := context.Background()
	if err := lazy.Push(ctx, "Urgent: AWS account suspended"); err == nil {
		t.Fatal("first push succeeded although the bot could not be created")
	}
	for _, msg := range []string{"Urgent: AWS account suspended", "Urgent: server down"} {
		if err := lazy.Push(ctx, msg); err != nil {
			t.Fatalf("push %q: %v", msg, err)
		}
	}
	if created != 2 || len(real.messages) != 2 {
		t.Fatalf("created %d times, delivered %v", created, real.messages)
	}
}
