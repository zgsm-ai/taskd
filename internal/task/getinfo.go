package task

import (
	"time"
)

/**
 * Information about queued/running tasks
 */
type TaskInstanceSummary struct {
	UUID        string            `json:"uuid"`                   //task unique ID
	Name        string            `json:"name"`                   //task name
	Status      string            `json:"status"`                 //task status
	CreatedBy   string            `json:"created_by"`             //created by
	Pool        string            `json:"pool"`                   //running pool
	Warning     string            `json:"warning"`                //warning message
	Error       string            `json:"error"`                  //error message
	Tags        map[string]string `json:"tags"`                   //tags
	CreateTime  *time.Time        `json:"create_time,omitempty"`  //queue time
	StartTime   *time.Time        `json:"start_time,omitempty"`   //start time
	RunningTime *time.Time        `json:"running_time,omitempty"` //actual running start time
	EndTime     *time.Time        `json:"end_time,omitempty"`     //end time
}

func (ti *TaskInstance) GetSummary() TaskInstanceSummary {
	return TaskInstanceSummary{
		UUID:        ti.UUID,
		Name:        ti.Name,
		Status:      ti.Status,
		Pool:        ti.Pool,
		Warning:     ti.Warning,
		Error:       ti.Error,
		Tags:        ti.GetTags(),
		CreatedBy:   ti.CreatedBy,
		CreateTime:  ti.CreateTime,
		StartTime:   ti.StartTime,
		RunningTime: ti.RunningTime,
		EndTime:     ti.EndTime,
	}
}
