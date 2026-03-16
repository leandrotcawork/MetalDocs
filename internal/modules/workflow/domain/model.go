package domain

type TransitionCommand struct {
	DocumentID string
	ToStatus   string
	ActorID    string
	Reason     string
	TraceID    string
}

type TransitionResult struct {
	DocumentID string
	FromStatus string
	ToStatus   string
}
