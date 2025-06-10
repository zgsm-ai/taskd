/**
 * Scheduler: queue management for pending tasks
 */
package flow

import (
	"fmt"
	"taskd/internal/task"
	"time"
)

/**
 * Notify when tasks are waiting to run
 */
func handleWaitingChan(tp *task.TaskPool) {
	for {
		job := <-tp.WaitingChan
		tp.PushWaitingJob(job)
		if _, running := tp.GetCapacity(); running > 0 {
			tp.SendRunningChan(1)
		}
	}
}

/**
 * Notify runner to start waiting tasks
 */
func handleRunningChan(tp *task.TaskPool) {
	for {
		count := <-tp.RunningChan // Number of tasks to start
		for i := 0; i < count; i++ {
			resumeWaitingJob(tp)
		}
	}
}

/**
 * Handle completed task instances
 */
func handleFinishedChan(tp *task.TaskPool) {
	for {
		job := <-tp.FinishedChan
		dealFinishedJob(job)
	}
}

/**
 * Handle all queued waiting tasks (mainly timeout processing)
 */
func handleWaitingJobs(tp *task.TaskPool) {
	for {
		<-time.After(1 * time.Second)
		tp.ForeachWaiting(func(job task.TaskJob) error {
			ti := job.Instance()
			begTime, maxDuration := ti.GetPhaseTime()
			if time.Since(begTime) >= maxDuration { // Queue timeout
				stopJob(job, task.TaskStatusKilled,
					fmt.Errorf("%s phase timeout: %v", ti.Phase().String(), maxDuration))
			}
			return nil
		})
	}
}

/**
 * Poll running tasks to advance task state machine
 */
func handleRunningJobs(tp *task.TaskPool) {
	for {
		<-time.After(1 * time.Second)
		tp.ForeachRunning(func(job task.TaskJob) error {
			if !job.Instance().GetStatus().IsFinished() {
				dealRunningJob(job)
				resolveMetrics(job.CustomMetrics())
			} else {
				tp.SendFinishedChan(job)
			}
			return nil
		})
	}
}
