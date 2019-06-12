package jobs

import (
	"context"

	"github.com/kscout/serverless-registry-api/config"

	"github.com/Noah-Huppert/golog"
	"github.com/google/go-github/v26/github"
	"go.mongodb.org/mongo-driver/mongo"
)

// JobTypeT is used to specify what type of job to start
type JobTypeT string

// JobTypeUpdateApps identifies a job of type UpdateApps
var JobTypeUpdateApps JobTypeT = "update_apps"

// JobTypeValidate identifies a job of type Validate
var JobTypeValidate JobTypeT = "validate"

// JobStartRequest provides informtion required to start a job
type JobStartRequest struct {
	// Type of job to start
	Type JobTypeT

	// Data required to start job
	Data []byte
}

// JobRunner manages starting jobs and shutting down gracefully
type JobRunner struct {
	// queue is a channel to which requests to start jobs are sent
	queue chan JobStartRequest

	// jobInstances holds jobs which can be run
	jobInstances map[JobTypeT]Job

	// Ctx
	Ctx context.Context

	// Logger
	Logger golog.Logger

	// Cfg is the server configuration
	Cfg *config.Config

	// GH is a GitHub API client
	GH *github.Client

	// MDbApps is used to access the apps collection
	MDbApps *mongo.Collection
}

// Init initializes a JobRunner. The Submit() and Run() methods will not work properly
// unless this method is called.
func (r *JobRunner) Init() {
	r.queue = make(chan JobStartRequest)

	r.jobInstances = map[JobTypeT]Job{}
	r.jobInstances[JobTypeUpdateApps] = UpdateAppsJob{
		Ctx: r.Ctx,
		Cfg: r.Cfg,
		GH: r.GH,
		MDbApps: r.MDbApps,
	}
	r.jobInstances[JobTypeValidate] = ValidateJob{
		Ctx: r.Ctx,
		Cfg: r.Cfg,
		GH: r.GH,
	}
}

// Submit new job
func (r JobRunner) Submit(req JobStartRequest) {
	r.queue <- req
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
			job, ok := r.jobInstances[req.Type]
			if !ok {
				r.Logger.Fatalf("cannot handle job type: %s", req.Type)
			}
			
			if err := job.Do(req.Data); err != nil {
				r.Logger.Errorf("failed to run %s job: %s",
					req.Type, err.Error())
			}
		}
	}
}
