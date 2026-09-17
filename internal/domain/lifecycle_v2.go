package domain

// Lifecycle is the Agent-neutral top-level task lifecycle exposed by command-result/v2 and context/v1.
type Lifecycle string

const (
	LifecycleNeedsAnalysis Lifecycle = "NEEDS_ANALYSIS"
	LifecycleReady         Lifecycle = "READY"
	LifecycleInProgress    Lifecycle = "IN_PROGRESS"
	LifecyclePaused        Lifecycle = "PAUSED"
	LifecycleWaiting       Lifecycle = "WAITING"
	LifecycleDeferred      Lifecycle = "DEFERRED"
	LifecycleBlocked       Lifecycle = "BLOCKED"
	LifecycleCompleted     Lifecycle = "COMPLETED"
	LifecycleCancelled     Lifecycle = "CANCELLED"
)

var LifecycleValues = []Lifecycle{
	LifecycleNeedsAnalysis,
	LifecycleReady,
	LifecycleInProgress,
	LifecyclePaused,
	LifecycleWaiting,
	LifecycleDeferred,
	LifecycleBlocked,
	LifecycleCompleted,
	LifecycleCancelled,
}

// ProjectLifecycle maps the v1 state machine into the v2 top-level lifecycle without rewriting history.
func ProjectLifecycle(state WorkflowState) Lifecycle {
	switch state {
	case StateReady:
		return LifecycleReady
	case StateInProgress, StateReview, StateFix:
		return LifecycleInProgress
	case StateBlocked:
		return LifecycleBlocked
	case StateDone:
		return LifecycleCompleted
	default:
		return LifecycleNeedsAnalysis
	}
}

func ProjectPhase(state WorkflowState) string {
	switch state {
	case StateReview:
		return "REVIEW"
	case StateFix:
		return "FIX"
	case StateInProgress:
		return "IMPLEMENTATION"
	default:
		return ""
	}
}
