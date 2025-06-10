package task

import (
	"time"
)

/**
 * Timeout settings in minutes
 */
type TimeoutSetting struct {
	Queue   int `json:"queue,omitempty"`   // Queue phase duration: Max time between task submission and invocation
	Init    int `json:"init,omitempty"`    // Init phase duration: Max time between task invocation and initialization completion
	Running int `json:"running,omitempty"` // Running phase duration: Time from initialization complete to task end
	Whole   int `json:"whole,omitempty"`   // Total duration: Max time from submission to overall completion
}

/**
 * Default timeout can be modified by system configuration
 */
var defaultTimeout Timeout = Timeout{
	Queue:   time.Hour * 24,       // Default queue phase allows 1 day, otherwise too long
	Init:    time.Hour * 24 * 7,   // Default init phase allows 1 day, prolonged init indicates system issues
	Running: time.Hour * 24 * 365, // Default maximum run time is 1 year
	Whole:   time.Hour * 24 * 365, // Default maximum run time is 1 year
}

/**
 *	Set default timeout values
 */
func SetDefaultTimeout(interval TimeoutSetting) {
	if interval.Queue != 0 {
		defaultTimeout.Queue = time.Duration(interval.Queue) * time.Minute
	}
	if interval.Init != 0 {
		defaultTimeout.Init = time.Duration(interval.Init) * time.Minute
	}
	if interval.Running != 0 {
		defaultTimeout.Running = time.Duration(interval.Running) * time.Minute
	}
	if interval.Whole != 0 {
		defaultTimeout.Whole = time.Duration(interval.Whole) * time.Minute
	}
}
