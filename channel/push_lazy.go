package channel

import (
	"context"
	"sync"
)

type lazyPusher struct {
	mu     sync.Mutex
	create func() (Pusher, error)
	pusher Pusher
}

func NewLazyPusher(create func() (Pusher, error)) Pusher {
	return &lazyPusher{create: create}
}

func (l *lazyPusher) Push(ctx context.Context, message string) error {
	l.mu.Lock()
	if l.pusher == nil {
		p, err := l.create()
		if err != nil {
			l.mu.Unlock()
			return err
		}
		l.pusher = p
	}
	p := l.pusher
	l.mu.Unlock()
	return p.Push(ctx, message)
}
