package tools

type AutoApproveInteractor struct{}

var _ Interactor = AutoApproveInteractor{}

func (AutoApproveInteractor) Notify(_ string) error                  { return nil }
func (AutoApproveInteractor) NotifyLink(_, _ string) error           { return nil }
func (AutoApproveInteractor) RequestApproval(_ string) (bool, error) { return true, nil }
