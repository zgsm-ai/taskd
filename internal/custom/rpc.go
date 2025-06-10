package custom

import (
	"context"
	"fmt"
	"io"
	"strings"

	"taskd/dao"
	"taskd/internal/task"
	"taskd/internal/utils"
)

// RPC task struct (using RESTful API requests)
type Rpc struct {
	ctx               context.Context   // Execution context
	url               string            // Request URL
	api               string            // API path
	method            string            // HTTP method
	headers           map[string]string // Request headers
	paths             map[string]string // Path parameters
	queries           map[string]string // Query parameters
	body              string            // Request body
	logs              []string          // Log messages
	ss                *utils.Session    // Session connection to web service
	task.TaskInstance                   // TaskInstance as a base class
}

/*
Example args JSON:
{
	"url": "http://127.0.0.1:8080",
	"method":"GET",
	"api": "/api/v1/namespaces/{namespace}/pods",
	"headers": {
		"Content-Type": "application/json"
	},
	"paths": {
		"namespace": "default"
	},
	"queries": {
		"labelSelector": "app=nginx"
	},
	"body": "{\"apiVersion\": \"v1\", \"kind\": \"Pod\", \"metadata\": {\"name\": \"nginx\"}, \"spec\": {\"containers\": [{\"name\": \"nginx\", \"image\": \"nginx:1.14.2\"}]}}"
}
*/

/**
 * Initialize task instance executed as native Pod
 */
func NewRpc(td *dao.TemplateRec, tr *dao.TaskRec) (task.TaskJob, error) {
	rpc := &Rpc{}
	if err := rpc.Init(td, tr); err != nil {
		return nil, fmt.Errorf("error in NewRpc init: %v", err)
	}
	extra, err := rpc.GetExtra()
	if err != nil {
		return nil, fmt.Errorf("error in NewRpc parse extra: %v", err)
	}
	args, err := task.ParseArgs(tr.Args)
	if err != nil {
		return nil, fmt.Errorf("error in NewRpc parse args: %v", err)
	}
	rpc.ctx = context.Background()

	rpc.url = task.GetArgString(extra, "url", "http://localhost:8080")
	rpc.api = task.GetArgString(extra, "api", "")
	rpc.method = task.GetArgString(extra, "method", "GET")
	rpc.headers = task.GetArgKvs(extra, "headers")
	rpc.body = task.GetArgString(args, "body", "")
	rpc.paths = task.GetArgKvs(args, "paths")
	rpc.queries = task.GetArgKvs(args, "queries")

	rpc.ss = utils.NewSession(rpc.url)
	return rpc, nil
}

/**
 * Start the task
 */
func (s *Rpc) Start() error {
	s.SetStatus(task.TaskStatusInit)
	s.Runner().OnJobStart(s)
	go func() {
		s.SetStatus(task.TaskStatusRunning)
		s.Runner().OnJobRunning(s)
		_, err := s.ss.Request(s.method, s.api, s.paths, s.queries, s.headers, []byte(s.body))
		if err != nil {
			s.SetError(task.TaskStatusFailed, err)
		} else {
			s.SetStatus(task.TaskStatusSucceeded)
		}
		s.Runner().OnJobEnd(s)
	}()
	return nil
}

/**
 * Fetch/detect current task status (composite status from all PODs)
 */
func (s *Rpc) FetchStatus() task.TaskStatus {
	return task.TaskStatus(s.Status)
}

/**
 * Continuously output logs in follow mode
 */
func (s *Rpc) FollowLogs(podName string, timestamp bool, tail int64) (io.ReadCloser, error) {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		for _, log := range s.logs {
			_, err := fmt.Fprintln(pw, log)
			if err != nil {
				// Stop on write failure
				return
			}
		}
	}()

	return pr, nil
}

/**
 * Get logs of Pods launched by the task
 */
func (s *Rpc) Logs(podName string, tail int64) ([]task.EntityLogs, error) {
	var results []task.EntityLogs
	var logs task.EntityLogs
	logs.Entity = ""
	logs.Logs = strings.Join(s.logs, "\n")
	logs.Completed = (s.Phase() == task.PhaseFinished)
	results = append(results, logs)
	return results, nil
}

/**
 * Stop the task
 */
func (s *Rpc) Stop() error {
	return nil
}

/**
 * Get task metrics
 */
func (s *Rpc) CustomMetrics() *task.Metric {
	return nil
}

/**
 * JOB type (different JOB types mean different underlying implementations)
 */
func (s *Rpc) Engine() task.TaskEngineKind {
	return task.RpcEngine
}
