package task

import (
	"fmt"
	"sync"
	"taskd/dao"
	"taskd/internal/utils"
)

/**
 *	Resource allocation manager
 *	Tracks resource totals and allocations, controls remaining allocatable resources
 */
type ResourceAlloc struct {
	Name     string
	Capacity utils.Quantity
	Allocate utils.Quantity
}

/**
 * Task pool summary
 */
type TaskPoolSummary struct {
	PoolId     string `json:"pool_id"`     // Pool identifier
	Engine     string `json:"engine"`      // Task engine used by pool
	Config     string `json:"config"`      // Pool configuration
	MaxWaiting int    `json:"max_waiting"` // Maximum queued tasks
	MaxRunning int    `json:"max_running"` // Maximum concurrent tasks
	Waiting    int    `json:"waiting"`     // Number of tasks currently waiting in pool
	Running    int    `json:"running"`     // Number of tasks currently running in pool
}

/**
 *	Resource information entry
 */
type ResourceItem struct {
	Name     string `json:"name"`     //resource name
	Capacity string `json:"capacity"` //configured capacity
	Allocate string `json:"allocate"` //allocated amount
	Remain   string `json:"remain"`   //actual remaining
}

/**
 *	Task pool details
 */
type TaskPoolDetail struct {
	PoolId     string                `json:"pool_id"`             // Pool ID
	Engine     string                `json:"engine"`              //task pool engine
	Config     string                `json:"config"`              //task pool configuration
	MaxWaiting int                   `json:"max_waiting"`         //maximum waiting tasks
	MaxRunning int                   `json:"max_running"`         //maximum parallel tasks
	Waiting    int                   `json:"waiting"`             //current waiting tasks
	Running    int                   `json:"running"`             //current running tasks
	Tasks      []TaskInstanceSummary `json:"tasks,omitempty"`     //task information in queue or running
	Resources  []ResourceItem        `json:"resources,omitempty"` //resource information of the pool
}

/**
 * Task pool
 */
type TaskPool struct {
	dao.Pool
	Runner       Runner                   // Pool runner
	Extension    any                      // Pool extension resources
	WaitingChan  chan TaskJob             // Channel for waiting tasks
	RunningChan  chan int                 // Channel to notify runner to start new tasks
	FinishedChan chan TaskJob             // Channel for finished tasks
	resources    map[string]ResourceAlloc // Resource allocation table
	runnings     map[string]TaskJob       // Running table
	waitings     []TaskJob                // Waiting queue
	locker       sync.RWMutex             // Read-write lock
}

/**
 *	Create a task pool
 */
func (tp *TaskPool) Init(pool *dao.Pool) {
	tp.Pool = *pool
	tp.runnings = make(map[string]TaskJob)
	tp.resources = make(map[string]ResourceAlloc)
	tp.WaitingChan = make(chan TaskJob, tp.Waiting)
	tp.RunningChan = make(chan int)
	tp.FinishedChan = make(chan TaskJob, tp.Running)
}

/**
 *	Send to waiting tasks channel
 */
func (tp *TaskPool) SendWaitingChan(job TaskJob) {
	tp.WaitingChan <- job
}

/**
 *	Send to running tasks channel
 */
func (tp *TaskPool) SendRunningChan(count int) {
	tp.RunningChan <- count
}

/**
 * Send to finished tasks channel
 */
func (tp *TaskPool) SendFinishedChan(job TaskJob) {
	tp.FinishedChan <- job
}

/**
 *	Get count of currently running tasks
 */
func (tp *TaskPool) GetRunningCount() int {
	return len(tp.runnings)
}

/**
 * Get count of currently waiting tasks
 */
func (tp *TaskPool) GetWaitingCount() int {
	return len(tp.waitings)
}

/**
 *	Iterate through waiting tasks
 */
func (tp *TaskPool) ForeachWaiting(handleJob func(job TaskJob) error) error {
	tp.locker.RLock()
	defer tp.locker.RUnlock()

	for _, job := range tp.waitings {
		if err := handleJob(job); err != nil {
			return err
		}
	}
	return nil
}

/**
 *	Iterate through running tasks
 */
func (tp *TaskPool) ForeachRunning(handleJob func(job TaskJob) error) error {
	tp.locker.RLock()
	defer tp.locker.RUnlock()

	for _, job := range tp.runnings {
		if err := handleJob(job); err != nil {
			return err
		}
	}
	return nil
}

/**
 *	Get capacity information
 */
func (tp *TaskPool) GetCapacity() (waiting, running int) {
	return tp.Waiting - len(tp.waitings), tp.Running - len(tp.runnings)
}

/**
 *	Load task pool resources
 */
func (tp *TaskPool) LoadResources() error {
	rcs, err := dao.ListPoolResources(tp.PoolId)
	if err != nil {
		return err
	}
	if len(rcs) == 0 {
		return nil
	}
	for _, rc := range rcs {
		var rhs utils.Quantity
		if err := rhs.Parse(rc.ResNum); err != nil {
			utils.Errorf("任务池[%s]的资源设置[%s=%s]有错误", rc.PoolId, rc.ResName, rc.ResNum)
		}
		tp.resources[rc.ResName] = ResourceAlloc{
			Name:     rc.ResName,
			Capacity: rhs,
		}
	}
	return nil
}

/**
 *	Reload resources (usually due to scaling), meaning the pool can handle more/fewer concurrent tasks
 */
func (tp *TaskPool) ReloadResources() error {
	if err := tp.LoadResources(); err != nil {
		return err
	}
	// Recalculate allocated resources
	for _, job := range tp.runnings {
		quotas := job.Instance().GetQuotas()
		for _, q := range quotas {
			if alloc, ok := tp.resources[q.ResName]; ok {
				alloc.Allocate.Plus(utils.Quantity{Amend: q.ResNum, Unit: q.ResFmt})
				tp.resources[q.ResName] = alloc
			}
		}
	}
	return nil
}

/**
 *	Get pool summary
 */
func (tp *TaskPool) GetSummary() TaskPoolSummary {
	var result TaskPoolSummary
	result.PoolId = tp.PoolId
	result.Config = tp.Config
	result.Engine = tp.Engine
	result.MaxRunning = tp.Running
	result.MaxWaiting = tp.Waiting
	result.Running = len(tp.runnings)
	result.Waiting = len(tp.waitings)
	return result
}

/**
 *	Get pool details
 */
func (tp *TaskPool) GetDetail() TaskPoolDetail {
	var result TaskPoolDetail
	result.PoolId = tp.PoolId
	result.Engine = tp.Engine
	result.Config = tp.Config
	result.MaxRunning = tp.Running
	result.MaxWaiting = tp.Waiting

	tp.locker.RLock()
	defer tp.locker.RUnlock()

	result.Running = len(tp.runnings)
	result.Waiting = len(tp.waitings)
	for _, job := range tp.runnings {
		result.Tasks = append(result.Tasks, job.Instance().GetSummary())
	}
	for _, job := range tp.waitings {
		result.Tasks = append(result.Tasks, job.Instance().GetSummary())
	}
	return result
}

/**
 *	Remove specified task from pool
 *	Remove matched task instance from running or waiting queue
 *	@param job TaskJob task object to remove
 */
func (tp *TaskPool) RemoveJob(job TaskJob) error {
	if job == nil {
		return nil
	}
	tp.locker.Lock()
	defer tp.locker.Unlock()

	ti := job.Instance()
	// Remove from running queue
	if _, exists := tp.runnings[ti.UUID]; exists {
		delete(tp.runnings, ti.UUID)
		return nil
	}

	// Remove from waiting queue
	for i, j := range tp.waitings {
		if j.Instance().UUID == ti.UUID {
			// Remove element using slice operation
			tp.waitings = append(tp.waitings[:i], tp.waitings[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("任务[%s]不在当前任务池中", ti.UUID)
}

/**
 *	Add task instance to task pool for execution
 */
func (tp *TaskPool) AddRunningJob(job TaskJob) {
	tp.locker.Lock()
	tp.runnings[job.Instance().UUID] = job
	tp.locker.Unlock()
}

/**
 *	Remove task instance from monitoring list
 */
func (tp *TaskPool) RemoveRunningJob(job TaskJob) {
	tp.locker.Lock()
	delete(tp.runnings, job.Instance().UUID)
	tp.locker.Unlock()
}

/**
 * Add task to waiting queue
 */
func (tp *TaskPool) PushWaitingJob(job TaskJob) {
	tp.locker.Lock()
	tp.waitings = append(tp.waitings, job)
	tp.locker.Unlock()
}

/**
 *	Pop highest priority task matching filter
 *	Dequeue task to start running
 */
func (tp *TaskPool) PopWaitingJob() (TaskJob, error) {
	tp.locker.Lock()
	defer tp.locker.Unlock()

	if len(tp.waitings) == 0 {
		return nil, fmt.Errorf("not exist")
	}
	job := tp.waitings[0]
	tp.waitings = tp.waitings[1:]
	return job, nil
}

/**
 *	Allocate resource quota for specified task
 */
func (tp *TaskPool) AllocQuotas(quotas []dao.Quota) error {
	for n, q := range quotas {
		if rq, ok := tp.resources[q.ResName]; !ok {
			return fmt.Errorf("there is no resource [%s] in Pool [%s]", q.ResName, tp.PoolId)
		} else {
			qt := utils.Quantity{
				Amend: q.ResNum,
				Unit:  q.ResFmt,
			}

			if err := rq.Allocate.Plus(qt); err != nil {
				return err
			}
			ret, _ := utils.QuantityCompare(rq.Capacity, rq.Allocate)
			if ret < 0 {
				tp.FreeQuotas(quotas[:n])
				return fmt.Errorf("resource [%s] is insufficient in Pool [%s]", q.ResName, tp.PoolId)
			}
		}
	}
	return nil
}

/**
 *	Release resource quota
 */
func (tp *TaskPool) FreeQuotas(quotas []dao.Quota) error {
	for _, q := range quotas {
		if rq, ok := tp.resources[q.ResName]; !ok {
			return fmt.Errorf("there is no resource [%s] in Pool [%s]", q.ResName, tp.PoolId)
		} else {
			qt := utils.Quantity{
				Amend: q.ResNum,
				Unit:  q.ResFmt,
			}
			if err := rq.Allocate.Minus(qt); err != nil {
				return err
			}
		}
	}
	return nil
}
