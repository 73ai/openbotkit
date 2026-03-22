package telegram

// Interrupter allows the Poller to interrupt a running agent.
type Interrupter interface {
	IsAgentRunning() bool
	Kill() bool
	RunningDelegateTasks() []TaskSummary
	KillDelegateTask(id string) bool
}

// TaskSummary is a minimal view of a running delegate task.
type TaskSummary struct {
	ID   string
	Task string
}
