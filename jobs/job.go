package jobs

// Job is a piece of logic
type Job interface {
	// Do job. Data argument holds arbitrary data. It is up to each job
	// to define what data it takes.
	Do(data []byte) error
}
