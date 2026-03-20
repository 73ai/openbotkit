package daemon

type SyncSignal struct {
	Source string
}

type SyncNotifier struct {
	ch chan SyncSignal
}

func NewSyncNotifier() *SyncNotifier {
	return &SyncNotifier{ch: make(chan SyncSignal, 16)}
}

func (n *SyncNotifier) Notify(source string) {
	select {
	case n.ch <- SyncSignal{Source: source}:
	default:
	}
}

func (n *SyncNotifier) C() <-chan SyncSignal {
	return n.ch
}
