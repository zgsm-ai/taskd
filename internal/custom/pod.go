package custom

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"taskd/dao"
	"taskd/internal/task"
	"taskd/internal/utils"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var mus map[string]*sync.Mutex

// Pod represents a pod type task
type Pod struct {
	GetLabel          func(string) v1.ListOptions // Label selector for pods
	ctx               context.Context             // Execution context
	replicas          int                         // Replica count
	task.TaskInstance                             // Base instance
}

/**
 * Initialize task instance executed as native pods
 */
func NewPod(td *dao.TemplateRec, tr *dao.TaskRec) (task.TaskJob, error) {
	// Initialize global variables
	if mus == nil {
		mus = make(map[string]*sync.Mutex)
	}
	if _, ok := mus[tr.Namespace]; !ok {
		mus[tr.Namespace] = &sync.Mutex{}
	}

	pod := &Pod{}
	if err := pod.Init(td, tr); err != nil {
		return nil, fmt.Errorf("error in NewPod init: %v", err)
	}
	extra, err := task.ParseArgs(tr.Extra)
	if err != nil {
		return nil, fmt.Errorf("failed to parse task_obj.extra in NewPod: %v", err)
	}
	pod.replicas = task.GetArgInt(extra, "replicas", 1)
	pod.ctx = context.Background()
	pod.GetLabel = utils.GetTaskLabelSelector
	return pod, nil
}

/**
 * Get Kubernetes client
 */
func (s *Pod) getClientset() *kubernetes.Clientset {
	clientset, ok := s.GetPool().Extension.(*kubernetes.Clientset)
	if !ok {
		return nil
	}
	return clientset
}

/**
 * Start task
 */
func (s *Pod) Start() error {
	defer func() {
		mus[s.Namespace].Unlock()
	}()
	mus[s.Namespace].Lock()
	if ck := s.GetPool().Config; ck != "" {
		return utils.ApplyInference(s.Namespace, s.Template, s.YamlContent, ck)
	}
	return utils.Apply(s.Namespace, s.Template, s.YamlContent)
}

/**
 * Get pod list started by task
 */
func (s *Pod) Get() []*corev1.Pod {
	podList, err := s.getClientset().CoreV1().Pods(s.Namespace).
		List(s.ctx, s.GetLabel(s.UUID))
	if err != nil {
		utils.Errorf("Failed to get pod list for task[%s]: %v", s.Title(), err)
	}

	var result []*corev1.Pod
	for _, v := range podList.Items {
		result = append(result, &v)
	}

	return result
}

/**
 * Get/detect current task status by aggregating pod states
 */
func (s *Pod) FetchStatus() task.TaskStatus {
	return s.Statuses().Status()
}

/**
 * Get status of pods started by task
 */
func (s *Pod) Statuses() PodStatusSet {
	results := NewPodStatusSet()
	for _, pod := range s.Get() {
		podName := pod.GetName()
		podStatus, err := s.getClientset().CoreV1().Pods(s.Namespace).
			Get(s.ctx, podName, v1.GetOptions{})
		if err != nil {
			utils.Errorf("Failed to get status for pod(%s) in task[%s]: %v", podName, s.Title(), err)
			results.Add(podName, StatusUnknown)
		} else {
			results.Add(podName, PodStatus(podStatus.Status.Phase))
		}
	}
	for i := 0; i < s.replicas-len(results); i++ {
		results.Add("", StatusNotExist)
	}
	return results
}

/**
 * Continuously output logs in follow mode
 */
func (s *Pod) FollowLogs(podName string, timestamp bool, tail int64) (io.ReadCloser, error) {
	return utils.GetPodFollowLogs(s.Namespace, *s.CreateTime, s.UUID, uint(tail))
}

/**
 * Get logs produced by pod with name podName
 */
func (s *Pod) podLogs(podName string, tail int64) (task.EntityLogs, error) {
	var result string
	if podName == "" {
		podName = "merged"
		podLogs, err := utils.GetTaskLogs(s.Namespace, s.UUID, uint(tail))
		if err != nil {
			return task.EntityLogs{}, fmt.Errorf("failed to read logs for pod(%s): %v", podName, err)
		}

		for _, l := range podLogs {
			result = l.Line + "\n" + result
		}
	} else {
		podLog := s.getClientset().CoreV1().Pods(s.Namespace).
			GetLogs(podName, &corev1.PodLogOptions{
				TailLines: &tail,
			}).Do(s.ctx)
		if podLog.Error() != nil {
			return task.EntityLogs{}, fmt.Errorf("failed to read logs for pod(%s): %v", podName, podLog.Error())
		}
		result_byte, err := podLog.Raw()
		if err != nil {
			return task.EntityLogs{}, fmt.Errorf("failed to read logs for pod(%s): %v", podName, err)
		}
		result = string(result_byte)
	}

	return task.EntityLogs{
		Logs:      result,
		Entity:    podName,
		Completed: s.Phase() == task.PhaseFinished,
	}, nil
}

/**
 * Get logs of pods started by task
 */
func (s *Pod) Logs(podName string, tail int64) ([]task.EntityLogs, error) {
	var results []task.EntityLogs
	// When podName is not empty, directly get its logs
	if podName != "" {
		result, err := s.podLogs(podName, tail)
		if err != nil {
			utils.Errorf("pod任务[%s] %v", s.Title(), err)
			return results, err
		}
		results = append(results, result)
		return results, nil
	}

	// When podName is empty, query launched pod list
	pods := s.Get()
	// If pod list is empty, get logs via loki
	if len(pods) == 0 {
		result, err := s.podLogs("", tail)
		if err != nil {
			utils.Errorf("pod任务[%s] %v", s.Title(), err)
			return results, err
		}
		results = append(results, result)
		return results, nil
	}

	// If pod list is not empty, get logs for all pods
	var podLogErrMsgs = make([]string, 0, len(pods))
	for _, pod := range pods {
		result, err := s.podLogs(pod.GetName(), tail)
		if err != nil {
			podLogErrMsgs = append(podLogErrMsgs, err.Error())
			continue
		}
		results = append(results, result)
	}
	if len(podLogErrMsgs) > 0 {
		utils.Errorf("Pod task[%s] errors: %v", s.Title(), strings.Join(podLogErrMsgs, "\n "))
		if len(results) == 0 {
			return results, fmt.Errorf(strings.Join(podLogErrMsgs, "\n"))
		}
	}

	return results, nil
}

/**
 * Stop task
 */
func (s *Pod) Stop() error {
	defer func() {
		mus[s.Namespace].Unlock()
	}()
	mus[s.Namespace].Lock()

	// Invoke synchronous kubectl delete command
	if ck := s.GetPool().Config; ck != "" {
		return utils.DeleteSyncInference(s.Namespace, s.Template, s.YamlContent, ck)
	}
	return utils.DeleteSync(&utils.DestroyItem{
		YamlContent: s.YamlContent,
		Namespace:   s.Namespace,
		TaskType:    s.Template,
		Force:       true,
	})
}

func (s *Pod) CustomMetrics() *task.Metric {
	return nil
}

/**
 * Job type (different job types imply different underlying implementations)
 */
func (s *Pod) Engine() task.TaskEngineKind {
	return task.PodEngine
}
