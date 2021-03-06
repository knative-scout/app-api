package jobs

import (
	"context"
	"fmt"

	"github.com/kscout/serverless-registry-api/config"
	"github.com/kscout/serverless-registry-api/metrics"

	"github.com/Noah-Huppert/golog"
	"github.com/google/go-github/v26/github"
	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/mongo"
)

// JobTypeT is used to specify what type of job to start
type JobTypeT string

// Job types identify different jobs which can be run
const (
	JobTypeUpdateApps JobTypeT = "update_apps"
	JobTypeValidate            = "validate"
)

// JobStartRequest provides informtion required to start a job
type JobStartRequest struct {
	// Type of job to start
	Type JobTypeT

	// Data required to start job
	Data []byte

	// CompleteChan will close when the job has been completed. This
	// does not guarantee the job finished successfully
	CompleteChan chan interface{}
}

// JobRunner manages starting jobs and shutting down gracefully
type JobRunner struct {
	// queue is a channel to which requests to start jobs are sent
	queue chan *JobStartRequest

	// jobInstances holds jobs which can be run
	jobInstances map[JobTypeT]Job

	// Ctx
	Ctx context.Context

	// Logger
	Logger golog.Logger

	// Cfg is the server configuration
	Cfg *config.Config

	// Metrics holds internal Prometheus metrics recorders
	Metrics metrics.Metrics

	// GH is a GitHub API client
	GH *github.Client

	// MDbApps is used to access the apps collection
	MDbApps *mongo.Collection
}

// Init initializes a JobRunner. The Submit() and Run() methods will not work properly
// unless this method is called.
func (r *JobRunner) Init() {
	r.queue = make(chan *JobStartRequest)

	r.jobInstances = map[JobTypeT]Job{}
	r.jobInstances[JobTypeUpdateApps] = UpdateAppsJob{
		Ctx:     r.Ctx,
		Cfg:     r.Cfg,
		GH:      r.GH,
		MDbApps: r.MDbApps,
	}
	r.jobInstances[JobTypeValidate] = ValidateJob{
		Ctx:    r.Ctx,
		Logger: r.Logger.GetChild("job.validate"),
		Cfg:    r.Cfg,
		GH:     r.GH,
	}
}

// Submit new job
func (r JobRunner) Submit(t JobTypeT, data []byte) *JobStartRequest {
	req := JobStartRequest{
		Type:         t,
		Data:         data,
		CompleteChan: make(chan interface{}),
	}

	r.queue <- &req

	return &req
}

// Run reads requests off the Queue and starts go routines to run jobs
// If the JobRunner.Ctx is canceled JobRunner will stop accepting jobs and
// return when there are no more jobs running.
// Should be run in a goroutine b/c this method blocks to run jobs.
func (r JobRunner) Run() {
	for {
		select {
		case <-r.Ctx.Done():
			return

		case req := <-r.queue:
			// Pre-metrics
			durationTimer := r.Metrics.StartTimer()

			// Run job
			job, ok := r.jobInstances[req.Type]
			if !ok {
				r.Logger.Fatalf("cannot handle job type: %s", req.Type)
			}

			jobSuccessful := "1"

			if err := job.Do(req.Data); err != nil {
				r.Logger.Errorf("failed to run %s job: %s",
					req.Type, err.Error())

				jobSuccessful = "0"
			}

			close(req.CompleteChan)
			r.Logger.Debugf("ran %s job", req.Type)

			// Post-metrics
			durationTimer.Finish(r.Metrics.JobsRunDurationsMilliseconds.
				With(prometheus.Labels{
					"job_type":   fmt.Sprintf("%s", req.Type),
					"successful": jobSuccessful,
				}))
		}
	}
}
