package task

import "io"

// Meta task metadata related
type Meta interface {
	Engine() TaskEngineKind  // Task engine
	Instance() *TaskInstance // Return corresponding task instance
}

// Action task actions related
type Action interface {
	Start() error // Task startup
	Stop() error  // Terminate task with specified status code
}

// Metric task monitoring related
type Metrics interface {
	FetchStatus() TaskStatus                               // Get actual task status
	Logs(string, int64) ([]EntityLogs, error)              // Task logs
	FollowLogs(string, bool, int64) (io.ReadCloser, error) // Implement continuous log output
	CustomMetrics() *Metric                                // Custom metrics reporting, also reports to user platforms but we keep our own copy
}

// TaskJob job type tasks
type TaskJob interface {
	Meta    // Metadata
	Action  // Actions
	Metrics // Metrics reporting
}

/**
 *	Task pool runner (poller mode, reactor mode)
 */
type Runner interface {
	Pool() *TaskPool
	OnJobStart(job TaskJob)
	OnJobRunning(job TaskJob)
	OnJobEnd(job TaskJob)
	Run()
}
