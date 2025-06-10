package task

import (
	"fmt"
	"taskd/dao"
	"taskd/internal/utils"
)

/**
 *	Engine type
 */
type TaskEngineKind string

/**
 *	Initialize job instance for specified task engine
 */
type FnNewJob func(*dao.TemplateRec, *dao.TaskRec) (TaskJob, error)

/**
 *	Initialize job pool for specified task engine
 */
type FnInitExtension func(*TaskPool) error

/**
 *	Initialize job runner for specified task engine
 */
type FnNewRunner func(*TaskPool) Runner

/**
 *	Defines a task engine
 */
type TaskEngine struct {
	NewJob        FnNewJob
	InitExtension FnInitExtension
	NewRunner     FnNewRunner
}

/**
 *	Task engine registry
 */
type TaskEngineRegistry map[TaskEngineKind]TaskEngine

/**
 *	Task engine registry
 */
var taskEngines TaskEngineRegistry

/**
 *	Task engine names
 */
var (
	PodEngine   TaskEngineKind = "pod"   // Execute job in POD mode
	CrdEngine   TaskEngineKind = "crd"   // Custom task engines like PytorchJob
	KFJobEngine TaskEngineKind = "kfjob" // kubeflow training-operator XXJob
	RpcEngine   TaskEngineKind = "rpc"   // RPC task executed via Restful API
)

/**
 *	Register task engine
 */
func RegisterEngine(t TaskEngineKind, newJob FnNewJob, initExt FnInitExtension, newRunner FnNewRunner) {
	if taskEngines == nil {
		taskEngines = make(TaskEngineRegistry)
	}

	taskEngines[t] = TaskEngine{
		NewJob:        newJob,
		InitExtension: initExt,
		NewRunner:     newRunner,
	}
}

/**
 *	Initialize task pool for specified engine
 */
func NewPool(pool *dao.Pool) (*TaskPool, error) {
	if engine, ok := taskEngines[TaskEngineKind(pool.Engine)]; !ok {
		return nil, fmt.Errorf("engine [%s] used by POOL [%s] is not registered", pool.Engine, pool.PoolId)
	} else {
		tp := &TaskPool{}
		tp.Init(pool)
		if engine.InitExtension != nil {
			err := engine.InitExtension(tp)
			if err != nil {
				return nil, err
			}
		}
		tp.Runner = engine.NewRunner(tp)
		return tp, nil
	}
}

/**
 *	Create a job instance using specified engine with data from TaskRec
 */
func NewJob(td *dao.TemplateRec, tr *dao.TaskRec) (TaskJob, error) {
	if _, ok := taskEngines[TaskEngineKind(td.Engine)]; !ok {
		return nil, fmt.Errorf("engine [%s] has not been registered yet", td.Engine)
	}
	return taskEngines[TaskEngineKind(td.Engine)].NewJob(td, tr)
}

/**
 *	Create job instance from task configuration
 */
func CreateJob(tr *dao.TaskRec) (TaskJob, error) {
	td, err := dao.LoadTemplate(tr.Template)
	if err != nil {
		utils.Errorf("Task [%s:%s] LoadTemplate failed: %v", tr.Template, tr.UUID, err)
		return nil, err
	}
	job, err := NewJob(td, tr)
	if err != nil {
		utils.Errorf("Task [%s:%s] NewJob failed: %v", tr.Template, tr.UUID, err)
		return nil, err
	}
	return job, nil
}

/**
 *	Load and fully restore TaskJob from database
 */
func LoadJob(uuid string) (TaskJob, error) {
	ti, err := dao.LoadTask(uuid)
	if err != nil {
		return nil, err
	}
	return CreateJob(ti)
}
