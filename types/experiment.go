package types

// ExperimentStatus represents the lifecycle state of an experiment.
type ExperimentStatus string

const (
	ExprDraft     ExperimentStatus = "draft"
	ExprRunning   ExperimentStatus = "running"
	ExprCompleted ExperimentStatus = "completed"
	ExprAbandoned ExperimentStatus = "abandoned"
)

// ValidExperimentStatuses lists all valid experiment statuses.
var ValidExperimentStatuses = []string{"draft", "running", "completed", "abandoned"}

// ExperimentKind describes the type of experiment.
type ExperimentKind string

const (
	ExprKindInterview   ExperimentKind = "interview"
	ExprKindLandingPage ExperimentKind = "landing-page"
	ExprKindPrototype   ExperimentKind = "prototype"
	ExprKindConcierge   ExperimentKind = "concierge"
	ExprKindSurvey      ExperimentKind = "survey"
	ExprKindABTest      ExperimentKind = "ab-test"
	ExprKindMock        ExperimentKind = "mock"
	ExprKindOther       ExperimentKind = "other"
)

// ValidExperimentKinds lists all valid experiment kinds.
var ValidExperimentKinds = []string{
	"interview", "landing-page", "prototype", "concierge",
	"survey", "ab-test", "mock", "other",
}

// SignalType distinguishes categories of learning signals.
type SignalType string

const (
	SignalUser     SignalType = "user"
	SignalBehavior SignalType = "behavior"
	SignalMarket   SignalType = "market"
)

// ValidSignalTypes lists all valid signal types.
var ValidSignalTypes = []string{"user", "behavior", "market"}

// ValidationSignal records a specific signal observed during an experiment.
type ValidationSignal struct {
	Type       SignalType `json:"type"`
	Metric     string     `json:"metric"`
	Expected   string     `json:"expected"`
	Actual     string     `json:"actual"`
	Confidence string     `json:"confidence"`
	RecordedAt string     `json:"recorded_at"`
}

// ExperimentData represents a single experiment testing a hypothesis.
type ExperimentData struct {
	ID              string             `json:"id"`
	ProjectID       string             `json:"project_id"`
	HypothesisID    string             `json:"hypothesis_id"`
	CycleID         string             `json:"cycle_id,omitempty"`
	Title           string             `json:"title"`
	Kind            ExperimentKind     `json:"kind"`
	Question        string             `json:"question"`
	Method          string             `json:"method"`
	SuccessSignal   string             `json:"success_signal"`
	KillCondition   string             `json:"kill_condition"`
	Status          ExperimentStatus   `json:"status"`
	Signals         []ValidationSignal `json:"signals,omitempty"`
	Outcome         string             `json:"outcome,omitempty"`
	KillTriggered   bool               `json:"kill_triggered,omitempty"`
	SpawnedFeatures []string           `json:"spawned_features,omitempty"`
	Labels          []string           `json:"labels,omitempty"`
	Version         int64              `json:"version"`
	CreatedAt       string             `json:"created_at"`
	UpdatedAt       string             `json:"updated_at"`
}
