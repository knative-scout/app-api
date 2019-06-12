package jobs

// Job is a piece of logic
type Job interface {
	// Do actions
	Do(data []byte) error
}
