package flow

import (
	"taskd/internal/task"
)

/**
 * Polling mode task executor
 */
type Poller struct {
	taskPool *task.TaskPool
}

func NewPoller(taskPool *task.TaskPool) task.Runner {
	return &Poller{
		taskPool: taskPool,
	}
}

func (r *Poller) OnJobEnd(job task.TaskJob) {
	job.Instance().Update()
}

func (r *Poller) OnJobStart(job task.TaskJob) {
	job.Instance().Update()
}

func (r *Poller) OnJobRunning(job task.TaskJob) {
	job.Instance().Update()
}

func (r *Poller) Pool() *task.TaskPool {
	return r.taskPool
}

func (r *Poller) Run() {
	handleRunningJobs(r.taskPool)
}
