package task

import (
	"time"
)

/**
 * Task status
 */
type TaskStatus string

/**
 * Task phase
 */
type TaskPhase uint

/**
 * Timeout settings
 */
type Timeout struct {
	Queue   time.Duration `json:"queue,omitempty"`   // Max time from submission to dispatch (queuing phase)
	Init    time.Duration `json:"init,omitempty"`    // Max time from dispatch to initialization complete
	Running time.Duration `json:"running,omitempty"` // Max time from initialization complete to task completion
	Whole   time.Duration `json:"whole,omitempty"`   // Max time from submission to completion (entire workflow)
}

/**
 *	4 phases of task execution: queue, initialization, running, finished
 */
const (
	PhaseQueue    TaskPhase = 1 //queue
	PhaseInit     TaskPhase = 2 //initialization, if failed in this phase, task will re-queue
	PhaseRunning  TaskPhase = 3 //running
	PhaseFinished TaskPhase = 4 //finished
)

const (
	TaskStatusQueue     TaskStatus = "Queue"     //in queue
	TaskStatusInit      TaskStatus = "Init"      //initializing in K8S
	TaskStatusRunning   TaskStatus = "Running"   //running
	TaskStatusSucceeded TaskStatus = "Succeeded" //succeeded
	TaskStatusFailed    TaskStatus = "Failed"    //failed
	TaskStatusCancelled TaskStatus = "Cancelled" //cancelled by user
	TaskStatusKilled    TaskStatus = "Killed"    //terminated by system
)

/**
 *	Check if task status indicates completion
 */
func (s TaskStatus) IsFinished() bool {
	return s == TaskStatusSucceeded || s == TaskStatusFailed || s == TaskStatusCancelled || s == TaskStatusKilled
}

/**
 *	Get execution phase from task status
 */
func (s TaskStatus) Phase() TaskPhase {
	switch s {
	case TaskStatusQueue:
		return PhaseQueue
	case TaskStatusInit:
		return PhaseInit
	case TaskStatusRunning:
		return PhaseRunning
	case TaskStatusSucceeded, TaskStatusFailed, TaskStatusCancelled, TaskStatusKilled:
		return PhaseFinished
	default:
		return PhaseQueue
	}
}

/**
 *	Get next phase in workflow
 */
func (phase TaskPhase) NextPhase() TaskPhase {
	switch phase {
	case PhaseQueue:
		return PhaseInit
	case PhaseInit:
		return PhaseRunning
	case PhaseRunning:
		return PhaseFinished
	default:
		return PhaseFinished
	}
}

/**
 *	String representation of task phase
 */
func (phase TaskPhase) String() string {
	switch phase {
	case PhaseQueue:
		return "Queue"
	case PhaseInit:
		return "Init"
	case PhaseRunning:
		return "Running"
	case PhaseFinished:
		return "Finished"
	default:
		return "Unknown"
	}
}

/**
 * Get timeout duration for specified phase
 */
func (t Timeout) GetPhaseTime(s TaskPhase) time.Duration {
	switch s {
	case PhaseQueue:
		return t.Queue
	case PhaseInit:
		return t.Init
	case PhaseRunning:
		return t.Running
	case PhaseFinished:
		return t.Whole
	default:
		return t.Whole
	}
}
