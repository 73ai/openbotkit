package channel

import "context"

type Pusher interface {
	Push(ctx context.Context, message string) error
}
