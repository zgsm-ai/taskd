package flow

import "taskd/internal/task"

type JobEventKind string

const (
	JobEventStart   JobEventKind = "start"
	JobEventEnd     JobEventKind = "end"
	JobEventRunning JobEventKind = "running"
)

type JobEvent struct {
	Kind JobEventKind
	Job  task.TaskJob
}

/**
 * Event notification mode (reactor) task executor
 */
type Reactor struct {
	taskPool *task.TaskPool
	events   chan JobEvent
}

func NewReactor(taskPool *task.TaskPool) task.Runner {
	return &Reactor{
		taskPool: taskPool,
		events:   make(chan JobEvent, taskPool.Running*3),
	}
}

func (r *Reactor) OnJobEnd(job task.TaskJob) {
	r.events <- JobEvent{
		Kind: JobEventEnd,
		Job:  job,
	}
}

func (r *Reactor) OnJobStart(job task.TaskJob) {
	r.events <- JobEvent{
		Kind: JobEventStart,
		Job:  job,
	}
}

func (r *Reactor) OnJobRunning(job task.TaskJob) {
	r.events <- JobEvent{
		Kind: JobEventRunning,
		Job:  job,
	}
}

func (r *Reactor) Pool() *task.TaskPool {
	return r.taskPool
}

func (r *Reactor) Run() {
	// Missing timeout logic
	for {
		event := <-r.events
		switch event.Kind {
		case JobEventStart:
			event.Job.Instance().UpdateStatus(task.TaskStatusInit)
		case JobEventRunning:
			event.Job.Instance().UpdateStatus(task.TaskStatusRunning)
		case JobEventEnd:
			r.taskPool.SendFinishedChan(event.Job)
		}
	}
}
