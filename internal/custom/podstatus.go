package custom

import (
	"taskd/internal/task"
)

/**
 *	Pod Status
 */
type PodStatus string

/**
 * Pod Status Set
 */
type PodStatusSet map[string]PodStatus

/**
 *	POD Status Collection
 */
const (
	StatusNotExist  PodStatus = "NotExist"  // Not exist/dequeued
	StatusUnknown   PodStatus = "Unknown"   // Unknown      POD: Unknown
	StatusPending   PodStatus = "Pending"   // Pending      POD: Pending
	StatusRunning   PodStatus = "Running"   // Running	 POD: Running
	StatusFailed    PodStatus = "Failed"    // Failed      POD: Failed
	StatusSucceeded PodStatus = "Succeeded" // Succeeded      POD: Succeeded
)

/**
 *	Task execution phase corresponding to POD status
 */
func (s PodStatus) Phase() task.TaskPhase {
	switch s {
	case StatusNotExist, StatusUnknown, StatusPending:
		return task.PhaseInit
	case StatusRunning:
		return task.PhaseRunning
	default:
		return task.PhaseFinished
	}
}

/**
 * Collect status of a new POD
 */
func (s *PodStatusSet) Add(podName string, status PodStatus) {
	(*s)[podName] = status
}

/**
 *	Priority sequence of POD statuses
 */
var podstatus2seq map[PodStatus]int = map[PodStatus]int{
	"Queue":     1,
	"NotExist":  2,
	"Unknown":   3,
	"Pending":   4,
	"Running":   5,
	"Failed":    6,
	"Succeeded": 7,
}

/**
 *	The earliest status among all PODs, collective status depends on one with highest priority
 */
func (s PodStatusSet) earlier() PodStatus {
	if len(s) == 0 {
		return StatusUnknown
	}
	earlier := StatusSucceeded
	for _, si := range s {
		if podstatus2seq[earlier] > podstatus2seq[si] {
			earlier = si
		}
	}
	return earlier
}

/**
 *	Task status corresponding to POD set status
 */
func (s PodStatusSet) Status() task.TaskStatus {
	switch s.earlier() {
	case StatusPending:
		return task.TaskStatusQueue
	case StatusNotExist, StatusUnknown:
		return task.TaskStatusInit
	case StatusRunning:
		return task.TaskStatusRunning
	case StatusSucceeded:
		return task.TaskStatusSucceeded
	case StatusFailed:
		return task.TaskStatusFailed
	default:
		return task.TaskStatusCancelled
	}
}

/**
 * Create POD status set
 */
func NewPodStatusSet() PodStatusSet {
	return make(PodStatusSet)
}
