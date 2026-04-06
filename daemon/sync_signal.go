package daemon

import "sync"

type SyncSignal struct {
	Source string
	Data   any // event payload (e.g. []int64 for gmail new email IDs)
}

type SyncNotifier struct {
	mu   sync.RWMutex
	subs []chan SyncSignal
}

func NewSyncNotifier() *SyncNotifier {
	return &SyncNotifier{}
}

// Subscribe returns a new channel that receives all signals.
func (n *SyncNotifier) Subscribe() <-chan SyncSignal {
	ch := make(chan SyncSignal, 16)
	n.mu.Lock()
	n.subs = append(n.subs, ch)
	n.mu.Unlock()
	return ch
}

// Notify sends a signal with no data payload.
func (n *SyncNotifier) Notify(source string) {
	n.send(SyncSignal{Source: source})
}

// NotifyWithData sends a signal with an attached data payload.
func (n *SyncNotifier) NotifyWithData(source string, data any) {
	n.send(SyncSignal{Source: source, Data: data})
}

func (n *SyncNotifier) send(sig SyncSignal) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	for _, ch := range n.subs {
		select {
		case ch <- sig:
		default:
		}
	}
}

// Subscribe must be called once per consumer before the event loop starts.
// There is no Unsubscribe — the channel lives as long as the notifier.
