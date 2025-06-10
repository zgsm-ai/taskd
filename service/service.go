package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"taskd/dao"
	"taskd/internal/flow"
	"taskd/internal/task"
	"taskd/internal/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/**
 * Result of Commit API
 */
type TaskCommitResult struct {
	UUID string `json:"uuid"`
}

/**
 * Result of tasks/{uuid}/status API
 */
type TaskStatusResult struct {
	Name     string `json:"name,omitempty"`
	Template string `json:"template,omitempty"`
	Status   string `json:"status,omitempty"`
}

/**
 * Request parameters for tasks/{uuid}/logs API
 */
type TaskLogsArgs struct {
	Entity     string `form:"entity,omitempty"` // Entity that task may start multiple instances (e.g. threads, PODs)
	Tail       int64  `form:"tail,omitempty"`
	Follow     bool   `form:"follow,omitempty"`
	Timestamps bool   `form:"timestamps,omitempty"`
}

/**
 * Result of tasks/{uuid}/logs API
 */
type TaskLogsResult struct {
	UUID    string            `json:"uuid"`
	Status  string            `json:"status"`
	Entitys []task.EntityLogs `json:"entities,omitempty"`
}

/**
 * Result of TaskTags/TaskGetTags API
 */
type TaskTagsResult struct {
	Tags map[string]string `json:"tags"`
}

/**
 * Parameters for creating task pool
 */
type TaskPoolArgs struct {
	dao.Pool
	Resources []dao.PoolResource `json:"resources"`
}

/**
 * Result of creating task pool
 */
type TaskPoolResult struct {
	PoolId string `json:"pool_id"`
}

/**
 * Define a task template
 */
func AddTemplate(arg *dao.TemplateRec) error {
	arg.CreateTime = time.Now().Local()
	if err := arg.Store(); err != nil {
		utils.Errorf("Task template [%s] store failed: %v", arg.Name, err)
		return utils.RethrowError(http.StatusInternalServerError, err)
	}
	return nil
}

/**
 * Update task template
 */
func UpdateTemplate(req *dao.TemplateRec) error {
	var td *dao.TemplateRec
	var err error
	if td, err = dao.LoadTemplate(req.Name); err != nil {
		return err
	}
	if req.Title != "" {
		td.Title = req.Title
	}
	if req.Schema != "" {
		td.Schema = req.Schema
	}
	if req.Engine != "" {
		td.Engine = req.Engine
	}
	if req.Extra != "" {
		td.Extra = req.Extra
	}
	return td.Update()
}

/**
 * Delete a task template
 */
func DeleteTemplate(name string) error {
	// Check if template is referenced by running tasks
	_, err := dao.LoadTemplate(name)
	if err != nil {
		return utils.RethrowError(http.StatusInternalServerError, err)
	}
	// Delete template record
	td := &dao.TemplateRec{Name: name}
	if err := td.Delete(); err != nil {
		return utils.RethrowError(http.StatusInternalServerError, err)
	}
	return nil
}

/**
 * Add a task pool with associated resources
 */
func AddPool(arg *TaskPoolArgs) error {
	err := dao.DB.Transaction(func(tx *gorm.DB) error {
		exists, err := arg.Exists(tx)
		if err != nil {
			return err
		}
		if exists {
			return utils.NewHttpError(http.StatusBadRequest, os.ErrExist.Error())
		}
		if err := arg.Store(tx); err != nil {
			return err
		}
		for _, r := range arg.Resources {
			if err := r.Store(tx); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return err
}

/**
 * Update task pool definition
 */
func UpdatePool(req *TaskPoolArgs) error {
	var pool *dao.Pool
	var err error
	if pool, err = dao.LoadPool(req.PoolId); err != nil {
		return err
	}
	if req.Engine != "" {
		pool.Engine = req.Engine
	}
	if req.Config != "" {
		pool.Config = req.Config
	}
	if req.Engine != "" {
		pool.Engine = req.Engine
	}
	pool.Running = req.Waiting
	pool.Waiting = req.Waiting
	if err = pool.Update(dao.DB); err != nil {
		return err
	}
	flow.ReloadPoolConfigs(pool.PoolId)
	return nil
}

/**
 * Submit a new task
 */
func TaskCommit(to *dao.TaskObjRec) (TaskCommitResult, error) {
	if to.UUID == "" {
		to.UUID = uuid.New().String()
	} else {
		exist, err := dao.ExistTask(to.UUID)
		if err != nil {
			return TaskCommitResult{}, utils.RethrowError(http.StatusInternalServerError, err)
		}
		if exist {
			return TaskCommitResult{}, utils.NewHttpError(http.StatusBadRequest,
				fmt.Sprintf("Task [%s] already exists", to.UUID))
		}
	}
	now := time.Now().Local()
	ti := dao.TaskRec{
		TaskObjRec: *to,
	}
	ti.CreateTime = &now
	ti.UpdateTime = &now
	ti.Status = string(task.TaskStatusQueue)

	if err := ti.Create(); err != nil {
		utils.Errorf("Task [%s:%s] store failed: %v", ti.Template, ti.UUID, err)
		return TaskCommitResult{}, utils.RethrowError(http.StatusInternalServerError, err)
	}
	_, err := flow.PoolNewJob(&ti)
	if err != nil {
		utils.Errorf("Task [%s:%s] start failed: %v", ti.Template, ti.UUID, err)
		return TaskCommitResult{}, utils.RethrowError(http.StatusExpectationFailed, err)
	}

	return TaskCommitResult{
		UUID: to.UUID,
	}, nil
}

/**
 * Task status
 */
func TaskStatus(uuid string) (TaskStatusResult, error) {
	to, err := GetTask(uuid)
	if err != nil {
		return TaskStatusResult{}, utils.RethrowError(http.StatusBadRequest, err)
	}
	result := TaskStatusResult{
		Name:     to.Name,
		Status:   string(to.Status),
		Template: to.Template,
	}
	return result, err
}

/*
 * Get task log stream
 * @param uuid Task ID
 * @param args Log query parameters
 * @return io.ReadCloser Log stream reader
 * @return error Error object
 */
func TaskFollowLogs(uuid string, args *TaskLogsArgs) (io.ReadCloser, error) {
	job, err := flow.GetJob(uuid)
	if err != nil {
		return nil, utils.NewHttpError(http.StatusBadRequest, err.Error())
	}
	// Task instance status is not finished
	if !job.Instance().GetStatus().IsFinished() {
		return job.FollowLogs(args.Entity, args.Timestamps, args.Tail)
	}
	// Task instance status is finished
	logs := job.Instance().EndLog
	var results []task.EntityLogs
	if logs != "" {
		if err := json.Unmarshal([]byte(logs), &results); err != nil {
			return nil, err
		}
		if args.Entity != "" {
			logs = ""
			for _, r := range results {
				if r.Entity == args.Entity {
					logs = r.Logs
					break
				}
			}
		} else if len(results) > 0 {
			logs = results[0].Logs
		}
	}

	// Convert string to io.Reader interface
	reader := strings.NewReader(logs)
	// Convert io.Reader to io.ReadCloser interface
	return io.NopCloser(reader), nil
}

/*
 * Get task logs
 * @param uuid Task ID
 * @param args Log query parameters
 * @return *TaskLogsResult Log result
 * @return error Error object
 */
func TaskLogs(uuid string, args *TaskLogsArgs) (*TaskLogsResult, error) {
	job, err := flow.GetJob(uuid)
	if err != nil {
		return nil, utils.NewHttpError(http.StatusBadRequest, err.Error())
	}
	result := TaskLogsResult{
		UUID:   job.Instance().UUID,
		Status: string(job.Instance().Status),
	}
	// Task instance status is non-ending
	if !job.Instance().GetStatus().IsFinished() {
		result.Entitys, err = job.Logs(args.Entity, args.Tail)
		if err != nil {
			return nil, utils.NewHttpError(http.StatusInternalServerError, err.Error())
		}
		return &result, nil
	}
	// Task instance status is ending
	logs := job.Instance().EndLog
	var podLogs []task.EntityLogs
	if logs != "" {
		if err := json.Unmarshal([]byte(logs), &podLogs); err != nil {
			return nil, utils.NewHttpError(http.StatusInternalServerError, err.Error())
		}
		result.Entitys = podLogs
	}
	return &result, nil
}

/*
 * Stop specified task
 * @param uuid Task ID
 * @return error Error object
 */
func TaskStop(uuid string) error {
	return flow.CancelJob(uuid)
}

/**
 * Tag task instance
 */
func TaskTags(uuid string, tags map[string]string) (map[string]string, error) {
	job, err := flow.GetJob(uuid)
	if err != nil {
		return tags, utils.NewHttpError(http.StatusBadRequest, err.Error())
	}
	job.Instance().SetTags(tags)
	return job.Instance().GetTags(), nil
}

/**
 * Get task tags
 */
func GetTaskTags(uuid string) (*TaskTagsResult, error) {
	job, err := flow.GetJob(uuid)
	if err != nil {
		return nil, utils.NewHttpError(http.StatusBadRequest, err.Error())
	}
	return &TaskTagsResult{
		Tags: job.Instance().GetTags(),
	}, nil
}

/**
 * Get task object by UUID
 */
func GetTask(uuid string) (*dao.TaskRec, error) {
	rec, err := dao.LoadTask(uuid)
	if err != nil {
		return nil, err
	}

	return rec, nil
}

/**
 * Delete task pool and associated resources
 */
func DeletePool(poolId string) error {
	if err := flow.RemovePool(poolId); err != nil {
		return err
	}
	// Delete database record
	return dao.DeletePool(poolId)
}
