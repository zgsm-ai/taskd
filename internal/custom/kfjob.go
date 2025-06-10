package custom

import (
	"context"
	"fmt"
	"io"
	"strings"
	"taskd/dao"
	"taskd/internal/task"
	"taskd/internal/utils"

	// trainingClientset   "github.com/kubeflow/training-operator/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// KFJob types from kubeflow training-operator
type KFJob struct {
	replicas          int                         // Number of replicas
	crdGroup          string                      // CRD group for training-operator Job
	crdVersion        string                      // CRD version for training-operator Job
	crdKind           string                      // CRD kind for training-operator Job
	crdPlural         string                      // CRD plural form for training-operator Job
	getLabel          func(string) v1.ListOptions // Label selector function for Pods
	ctx               context.Context             // Context for operations
	task.TaskInstance                             // Base task instance
}

var jobKindsAccepted []string = []string{
	"PyTorchJob",
	"TFJob",
	"MPIJob",
}

func isAcceptedJobKind(jobKind string) bool {
	for _, t := range jobKindsAccepted {
		if t == jobKind {
			return true
		}
	}
	return false
}

/**
 * Initialize a KFJob task instance
 * @param td *task.TemplateRec Task template record
 * @param tr *task.TaskRec Task instance data
 */
func NewKFJob(td *dao.TemplateRec, tr *dao.TaskRec) (task.TaskJob, error) {
	crd := &KFJob{}

	if err := crd.Init(td, tr); err != nil {
		return nil, fmt.Errorf("error in NewKFJob init: %v", err)
	}
	extra, err := task.ParseArgs(tr.Extra)
	if err != nil {
		return nil, fmt.Errorf("error in NewKFJob parse task_obj.extra: %v", err)
	}
	tdargs, err := task.ParseArgs(td.Extra)
	if err != nil {
		return nil, fmt.Errorf("error in NewKFJob parse template.extra: %v", err)
	}
	crd.crdGroup = task.GetArgString(tdargs, "group", "kubeflow.org")
	crd.crdKind = task.GetArgString(tdargs, "kind", "PyTorchJob")
	crd.crdPlural = task.GetArgString(tdargs, "plural", "pytorchjobs")
	crd.crdVersion = task.GetArgString(tdargs, "version", "v1")
	if !isAcceptedJobKind(crd.crdKind) {
		utils.Errorf("任务[%s]:\n warning: NewKFJob: 'template.extra.kind' is not recommend: %s", crd.Title(), crd.crdKind)
	}

	crd.replicas = task.GetArgInt(extra, "masterNum", 1) + task.GetArgInt(extra, "workerNum", 0)
	crd.getLabel = utils.GetTaskLabelSelector
	crd.ctx = context.Background()

	return crd, nil
}

/**
 * Get Kubernetes clientset
 */
func (s *KFJob) getClientset() *kubernetes.Clientset {
	clientset, ok := s.GetPool().Extension.(*kubernetes.Clientset)
	if !ok {
		return nil
	}
	return clientset
}

/**
 * Start the task
 */
func (s *KFJob) Start() error {
	return utils.Apply(s.Namespace, s.Template, s.YamlContent)
}

/**
 * Get list of Pods for the task
 */
func (s *KFJob) Get() *corev1.PodList {
	podList, err := utils.Clientset.CoreV1().Pods(s.Namespace).
		List(s.ctx, s.getLabel(s.UUID))
	if err != nil {
		utils.Errorf("任务[%s]:\n 获取启动的Pod列表失败: %v", s.Title(), err)
	}
	return podList
}

/**
 * Initialize kubeflow-operator client
 */
// func initKFClientset() error {
// 	config, err := rest.InClusterConfig()
// 	if err != nil {
// 		flag.Parse()
// 		config, err = clientCmd.BuildConfigFromFlags("", *utils.KubeConfig)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	if KFClientset, err = dynamic.NewForConfig(config); err != nil {
// 		return err
// 	}
// 	return nil
// }

/**
 * Get latest status from Job's status.conditions
 */
func (s *KFJob) getJobStatus() (string, error) {
	return utils.GetPytorchJobStatus(s.Namespace, "pytorchjob", s.Name)
}

/**
 * Get list of events for the Job
 */
func (s *KFJob) getJobEvents() (string, error) {
	fieldSelector := fmt.Sprintf("involvedObject.namespace=%s,involvedObject.name=%s,involvedObject.kind=%s",
		s.Namespace, s.Name, s.crdKind)
	events, err := utils.Clientset.CoreV1().Events(s.Namespace).
		List(s.ctx, v1.ListOptions{
			FieldSelector: fieldSelector,
		})

	if err != nil {
		return "get events failed", fmt.Errorf("无法读取事件: %v", err)
	}
	var result string
	for _, event := range events.Items {
		result += fmt.Sprintf("Reason: %s, Message: %s\n", event.Reason, event.Message)
	}
	return result, nil
}

var jobStatus2TaskStatus map[string]task.TaskStatus = map[string]task.TaskStatus{
	"Created":   task.TaskStatusInit,
	"Running":   task.TaskStatusRunning,
	"Succeeded": task.TaskStatusSucceeded,
	"Failed":    task.TaskStatusFailed,
	"Unknown":   task.TaskStatusInit, //
}

/**
 * Get current job status and map to task status
 */
func (s *KFJob) FetchStatus() task.TaskStatus {
	status, err := s.getJobStatus()
	if err != nil {
		utils.Errorf("任务[%s]:\n getJobStatus error: %v", s.Title(), err)
		return task.TaskStatusInit //if any issue occurs, return an earlier state that will be ignored by the status handler
	}
	taskStatus, ok := jobStatus2TaskStatus[status]
	if !ok {
		return task.TaskStatusInit
	}
	return taskStatus
}

/**
 * Get continuous log stream
 */
func (s *KFJob) FollowLogs(podName string, timestamp bool, tail int64) (io.ReadCloser, error) {
	if podName == "" {
		for _, pod := range s.Get().Items {
			podName = pod.GetName()
		}
		if podName == "" {
			return nil, fmt.Errorf("no any pod")
		}
	}
	req := utils.Clientset.CoreV1().Pods(s.Namespace).
		GetLogs(podName, &corev1.PodLogOptions{
			Follow:     true,
			Timestamps: timestamp,
			TailLines:  &tail,
		})
	return req.Stream(context.Background())
}

/**
 * Get logs for the specified Pod
 */
func (s *KFJob) podLogs(podName string, tail int64) (task.EntityLogs, error) {
	var podLogs []utils.LogEntry
	var err error
	if podName == "" {
		podName = "merged"
		podLogs, err = utils.GetTaskLogs(s.Namespace, s.UUID, uint(tail))
	} else {
		podLogs, err = utils.GetPodLogs(s.Namespace, podName, *s.CreateTime, uint(tail))
	}

	if err != nil {
		return task.EntityLogs{}, fmt.Errorf("读取Pod(%s)的Loki日志失败: %v", podName, err)
	}

	result := ""
	for _, l := range podLogs {
		result += "\n" + l.Line
	}

	completed := false
	pod, err := s.getClientset().CoreV1().Pods(s.Namespace).
		Get(s.ctx, podName, v1.GetOptions{})
	if err == nil {
		podStatus := PodStatus(pod.Status.Phase)
		if podStatus.Phase() == task.PhaseFinished {
			completed = true
		}
	}

	return task.EntityLogs{
		Logs:      result,
		Entity:    podName,
		Completed: completed,
	}, nil
}

/**
 * Get logs for the task's Pods
 */
func (s *KFJob) Logs(podName string, tail int64) ([]task.EntityLogs, error) {
	var results []task.EntityLogs
	// When podName is not empty, directly get logs for podName
	if podName != "" {
		logs, err := s.podLogs(podName, tail)
		if err != nil {
			utils.Errorf("kfjob任务[%s] %v", s.Title(), err)
			return results, err
		}
		results = append(results, logs)
		return results, nil
	}

	// When podName is empty, get all pods
	items := s.Get().Items
	// If pod list is empty, get logs via loki
	if len(items) == 0 {
		logs, err := s.podLogs("", tail)
		if err != nil {
			utils.Errorf("kfjob任务[%s] %v", s.Title(), err)
			return results, err
		}
		results = append(results, logs)
		return results, nil
	}

	// If pod list is not empty, get logs for all pods
	var podLogErrMsgs = make([]string, 0, len(items))
	for _, pod := range items {
		logs, err := s.podLogs(pod.GetName(), tail)
		if err != nil {
			podLogErrMsgs = append(podLogErrMsgs, err.Error())
			continue
		}
		results = append(results, logs)
	}

	// Get job events list
	eventLog, err := s.getJobEvents()
	if err != nil {
		utils.Errorf("kfjob任务[%s] %v", s.Title(), err)
	}
	results = append(results, task.EntityLogs{
		Entity:    fmt.Sprintf("%s %s/%s events", s.crdKind, s.Namespace, s.Name),
		Logs:      eventLog,
		Completed: task.TaskStatus(s.Instance().Status).IsFinished(),
	})

	if len(podLogErrMsgs) > 0 {
		utils.Errorf("kfjob任务[%s] %v", s.Title(), strings.Join(podLogErrMsgs, "\n "))
		if len(results) == 1 {
			return results, fmt.Errorf(strings.Join(podLogErrMsgs, "\n"))
		}
	}
	return results, nil
}

/**
 * Stop the task
 */
func (s *KFJob) Stop() error {
	// Call async kubectl delete command
	utils.K8sDelete(s.YamlContent, true)
	return nil
}

func (s *KFJob) CustomMetrics() *task.Metric {
	return nil
}

/**
 * Job type (different job types indicate different underlying implementations)
 */
func (s *KFJob) Engine() task.TaskEngineKind {
	return task.KFJobEngine
}
