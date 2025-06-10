/**
 * Keeper: monitor for running tasks
 */
package flow

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"taskd/dao"
	"taskd/internal/task"
	"taskd/internal/utils"
	"time"
)

// Read-write lock
var allPoolsMutex sync.RWMutex
var allJobsMutex sync.RWMutex

/**
 * All task pools
 */
var allPools map[string]*task.TaskPool

/**
 * Lookup table for running tasks, indexed by task UUID
 */
var allJobs map[string]task.TaskJob

/**
 * Initialize event handler for task instances
 */
func Init() error {
	allJobs = make(map[string]task.TaskJob)
	allPools = make(map[string]*task.TaskPool)

	if err := initPools(); err != nil {
		return err
	}
	return nil
}

/**
 * Reload pool configurations
 */
func ReloadPoolConfigs(poolId string) error {
	for _, tp := range allPools {
		if poolId != tp.PoolId {
			continue
		}
		if err := tp.ReloadResources(); err != nil {
			utils.Errorf("Pool [%s] ReloadResources failed: %s", tp.PoolId, err.Error())
			return err
		}
	}
	return nil
}

/**
 * Reload unfinished task instances
 */
func ReloadHistoryTasks() error {
	tasks, err := dao.LoadTasks_NotFinished()
	if err != nil {
		return err
	}
	for _, tr := range tasks {
		_, err := PoolNewJob(&tr)
		if err != nil {
			utils.Errorf("Task [%s:%s] reload queue job err: %v", tr.Template, tr.UUID, err)
		} else {
			utils.Infof("Task [%s:%s] reload queue job", tr.Template, tr.UUID)
		}
	}
	return nil
}

/**
 * Initialize task pool module
 * Requires database module to be initialized first
 */
func initPools() error {
	pools, err := dao.ListPools()
	if err != nil {
		return err
	}
	for _, p := range pools {
		if err := initPool(p); err != nil {
			return err
		}
	}
	return nil
}

/**
 * Initialize a task pool
 */
func initPool(pool dao.Pool) error {
	tp, err := task.NewPool(&pool)
	if err != nil {
		return err
	}
	if err = tp.LoadResources(); err != nil {
		return fmt.Errorf("pool [%s] LoadResources failed: %s", pool.PoolId, err.Error())
	}
	allPoolsMutex.Lock()
	allPools[pool.PoolId] = tp
	allPoolsMutex.Unlock()

	go handleWaitingChan(tp)
	go handleRunningChan(tp)
	go handleWaitingJobs(tp)
	go tp.Runner.Run()
	go handleFinishedChan(tp)

	return nil
}

/**
 * Get information for all task pools
 */
func ListPools() []task.TaskPoolSummary {
	var pools []task.TaskPoolSummary
	for _, p := range allPools {
		pools = append(pools, p.GetSummary())
	}
	sort.Slice(pools, func(i, j int) bool {
		return pools[i].PoolId < pools[j].PoolId
	})
	return pools
}

/**
 * Get task pool by poolId
 */
func GetPool(poolId string) *task.TaskPool {
	allPoolsMutex.RLock()
	defer allPoolsMutex.RUnlock()
	return allPools[poolId]
}

/**
 * Remove a task pool
 */
func RemovePool(poolId string) error {
	allPoolsMutex.Lock()
	defer allPoolsMutex.Unlock()

	tp, ok := allPools[poolId]
	if !ok {
		return nil
	}

	// Check if there are running tasks
	if running := tp.GetRunningCount() + tp.GetWaitingCount(); running > 0 {
		return fmt.Errorf("pool [%s] has %d tasks", poolId, running)
	}

	delete(allPools, poolId)
	return nil
}

/**
 * Select the best pool to execute the job
 */
func selectPool(job task.TaskJob) (*task.TaskPool, error) {
	ti := job.Instance()

	allPoolsMutex.RLock()
	defer allPoolsMutex.RUnlock()

	if ti.Pool != "" {
		tp, ok := allPools[ti.Pool]
		if !ok {
			return nil, fmt.Errorf("pool [%s] is not exist", ti.Pool)
		}
		return tp, nil
	}
	var selected *task.TaskPool = nil
	var maxWaiting int = 0
	for _, tp := range allPools {
		if task.TaskEngineKind(tp.Engine) == job.Engine() {
			waiting, _ := tp.GetCapacity()
			if waiting > maxWaiting {
				maxWaiting = waiting
				selected = tp
			}
		}
	}
	if selected != nil {
		return selected, nil
	}
	return nil, fmt.Errorf("there is no pool available to run the [%s] engine required by task [%s]",
		job.Engine(), ti.Title())
}

/**
 * Find/reload TaskJob by instance ID
 */
func GetJob(uuid string) (task.TaskJob, error) {
	allJobsMutex.RLock()
	job, ok := allJobs[uuid]
	allJobsMutex.RUnlock()
	if ok {
		return job, nil
	}
	return task.LoadJob(uuid)
}

/**
 * Create a new job and add it to the pool for execution
 */
func PoolNewJob(tr *dao.TaskRec) (task.TaskJob, error) {
	job, err := task.CreateJob(tr)
	if err != nil {
		return nil, err
	}
	tp, err := selectPool(job)
	if err != nil {
		return nil, err
	}
	job.Instance().AttachPool(tp)

	allJobsMutex.Lock()
	allJobs[tr.UUID] = job
	allJobsMutex.Unlock()

	tp.SendWaitingChan(job)
	return job, err
}

/**
 * Cancel a task
 */
func CancelJob(uuid string) error {
	allJobsMutex.RLock()
	job, ok := allJobs[uuid]
	allJobsMutex.RUnlock()
	if !ok {
		exist, err := dao.ExistTask(uuid)
		if err != nil || !exist {
			return utils.NewHttpError(http.StatusBadRequest, fmt.Sprintf("task [%s] is not exist", uuid))
		}
		utils.Infof("Task [%s] has ended early", uuid)
		return nil
	}
	stopJob(job, task.TaskStatusCancelled, fmt.Errorf("user cancelled"))
	return nil
}

/**
 * Start a task instance
 */
func startJob(job task.TaskJob) error {
	ti := job.Instance()
	ti.Prerun()
	if err := job.Start(); err != nil {
		stopJob(job, task.TaskStatusFailed, err)
		return fmt.Errorf("task [%s] start failed: %v", ti.Title(), err)
	}
	ti.GetPool().AddRunningJob(job)
	return nil
}

/**
 * Request task instance to stop
 */
func stopJob(job task.TaskJob, status task.TaskStatus, err error) {
	if !status.IsFinished() {
		panic(fmt.Errorf("status [%s] is not completed", status))
	}
	ti := job.Instance()
	if err != nil {
		ti.SetError(status, err)
	} else {
		ti.SetStatus(status)
	}
	ti.GetPool().SendFinishedChan(job)
}

/**
 * Process jobs in the waiting queue
 */
func resumeWaitingJob(tp *task.TaskPool) {
	job, _ := tp.PopWaitingJob()
	if job == nil {
		return
	}
	err := startJob(job)
	if err != nil {
		utils.Errorf("Task [%s] start failed: %v", job.Instance().Title(), err)
	} else {
		utils.Infof("Task [%s] start succeeded", job.Instance().Title())
	}
}

/**
 * Handle state transitions for running task instances
 * CurrentStatus updated state value of the task
 */
func dealRunningJob(job task.TaskJob) {
	ti := job.Instance()
	timeout := ti.GetTimeout()
	status := job.FetchStatus()
	phase := status.Phase()
	if ti == nil {
		stopJob(job, task.TaskStatusFailed, fmt.Errorf("failed to get task instance:%v", timeout.Whole))
		return
	}
	if phase <= ti.Phase() { // Phase unchanged or state fetch incomplete causing phase miscalculation
		begTime, maxDuration := ti.GetPhaseTime()
		if time.Since(begTime) >= maxDuration { // Phase timeout
			stopJob(job, task.TaskStatusFailed,
				fmt.Errorf("%s phase execution exceeded limit:%v", ti.Phase().String(), maxDuration))
			return
		}
		if time.Since(*ti.StartTime) >= timeout.Whole { // Task overall timeout
			stopJob(job, task.TaskStatusFailed,
				fmt.Errorf("task execution exceeded total time limit:%v", timeout.Whole))
		}
		return
	}
	now := time.Now().Local()
	// Task execution phase changed - need to update status, phase, time etc.
	ti.UpdateStatus(status)
	if phase >= task.PhaseFinished { // Task completed
		utils.Infof("Task [%s] is finished, status: %v", ti.Title(), status)
		stopJob(job, status, nil)
		return
	} else if phase == task.PhaseRunning {
		ti.RunningTime = &now
	} else if phase == task.PhaseInit {
		ti.StartTime = &now
	}
	// Task overall timeout
	if time.Since(*ti.StartTime) >= timeout.Whole {
		stopJob(job, task.TaskStatusFailed,
			fmt.Errorf("task execution exceeds total time limit: %v", timeout.Whole))
	}
}

/**
 * Record final logs
 */
func updateEndlog(job task.TaskJob) {
	ti := job.Instance()
	endLog, err := job.Logs("", 200)
	if err != nil {
		utils.Errorf("Job [%s] fetch logs failed: %v", ti.Title(), err)
	}
	v, err := json.Marshal(endLog)
	if err != nil {
		utils.Errorf("Crd [%s] marshal logs failed:%v", ti.Title(), err)
	}
	ti.SetEndLog(string(v))
}

/**
 * Process completed tasks, including status updates and callbacks
 */
func dealFinishedJob(job task.TaskJob) {
	ti := job.Instance()
	if !ti.GetStatus().IsFinished() {
		panic(fmt.Sprintf("Task [%s] is not completed", ti.Title()))
	}
	// 1. Stop the task
	if err := job.Stop(); err != nil {
		utils.Errorf("Task [%s] stop failed: %s", ti.Title(), err)
	}
	// 2. Release quotas
	ti.FreeQuotas()
	// 3. Remove job from task pool
	tp := ti.GetPool()
	if tp != nil {
		tp.RemoveJob(job)
	}
	// 4. Notify pool to start a new job
	tp.SendRunningChan(1)
	// 5. Record final logs and clean up
	updateEndlog(job)
	// 6. Bury the remains
	ti.Bury()
	// 7. Notify next of kin
	ti.SendCallback(ti.GetError())

	allJobsMutex.Lock()
	delete(allJobs, ti.UUID)
	allJobsMutex.Unlock()
}

/**
 * Handle monitoring metrics reporting
 */
func resolveMetrics(*task.Metric) {
	// TODO:It can be reported to the monitoring and log modules later.

}
