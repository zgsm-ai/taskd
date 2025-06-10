package task

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"taskd/dao"
	"taskd/internal/utils"
	"text/template"
	"time"
)

/**
 * Log information for task entities (threads, PODs, etc.)
 */
type EntityLogs struct {
	Completed bool   `json:"completed"`
	Entity    string `json:"entity"`
	Logs      string `json:"logs"`
}

// Task instance
type TaskInstance struct {
	dao.TaskRec

	template *dao.TemplateRec  // Task template data
	pool     *TaskPool         // Associated task pool
	quotas   []dao.Quota       // Allocated resource quotas
	phase    TaskPhase         // Current phase
	tags     map[string]string // Tags
}

/**
 * Initialize task instance
 * Two initialization methods:
 * 1. Initialize based on task object
 * 2. Copy task instance data loaded from database
 * @param td *dao.TemplateRec Task class definition
 * @param tr *dao.TaskRec Task object
 */
func (ti *TaskInstance) Init(td *dao.TemplateRec, tr *dao.TaskRec) error {
	ti.TaskRuntimeRec = tr.TaskRuntimeRec
	ti.TaskObjRec = tr.TaskObjRec
	ti.template = td

	if ti.YamlContent != "" {
		return nil
	}
	now := time.Now().Local()
	ti.Status = string(TaskStatusQueue)
	ti.CreateTime = &now
	ti.UpdateTime = &now
	ti.phase = PhaseQueue
	if _, err := ti.Compile(); err != nil {
		return fmt.Errorf("task [%s] compile template failed: %s", ti.Title(), err.Error())
	}
	return nil
}

/**
 * Prepare task instance for running
 */
func (ti *TaskInstance) Prerun() {
	now := time.Now().Local()
	ti.Status = string(TaskStatusInit)
	ti.phase = PhaseInit
	ti.StartTime = &now
	ti.UpdateTime = &now
	ti.Update()
}

/**
 * Update status
 */
func (ti *TaskInstance) UpdateStatus(status TaskStatus) {
	now := time.Now().Local()
	ti.Status = string(status)
	ti.UpdateTime = &now
	ti.phase = status.Phase()

	if status == TaskStatusInit {
		ti.StartTime = &now
	} else if status == TaskStatusRunning {
		ti.RunningTime = &now
	} else if status.IsFinished() {
		ti.EndTime = &now
	}
	ti.Update()
}

/**
 * Runner used by task instance
 */
func (ti *TaskInstance) Runner() Runner {
	return ti.pool.Runner
}

/**
 * Get task pool containing this instance
 */
func (ti *TaskInstance) GetPool() *TaskPool {
	return ti.pool
}

/**
 * Attach task instance to a task pool
 * @param tp *TaskPool Task pool object
 */
func (ti *TaskInstance) AttachPool(tp *TaskPool) {
	ti.pool = tp
}

/**
 * Set warning message
 */
func (ti *TaskInstance) SetWarning(warning string) {
	ti.Warning = warning
}

/**
 * Errors occurred during job execution
 */
func (ti *TaskInstance) GetError() string {
	return ti.Error
}

/**
 * Set job execution error
 */
func (ti *TaskInstance) SetError(finished TaskStatus, e error) {
	if !finished.IsFinished() {
		panic(fmt.Errorf("status [%s] is not completed", finished))
	}
	ti.Status = string(finished)
	ti.Error = e.Error()
}

/**
 * Get timing information for current phase
 */
func (ti *TaskInstance) GetPhaseTime() (beg time.Time, maxDuration time.Duration) {
	switch ti.phase {
	case PhaseQueue:
		return *ti.CreateTime, ti.GetTimeout().Queue
	case PhaseInit:
		return *ti.StartTime, ti.GetTimeout().Init
	case PhaseRunning:
		return *ti.RunningTime, ti.GetTimeout().Running
	case PhaseFinished:
		return *ti.EndTime, ti.GetTimeout().Whole
	default:
		return *ti.EndTime, ti.GetTimeout().Whole
	}
}

/**
 * Implements TaskJob interface for introspection
 */
func (ti *TaskInstance) Instance() *TaskInstance {
	return ti
}

/**
 * Task title
 */
func (ti *TaskInstance) Title() string {
	return fmt.Sprintf("%s:%s", ti.Template, ti.UUID)
}

/**
 * Current phase
 */
func (ti *TaskInstance) Phase() TaskPhase {
	return ti.phase
}

/**
 * Get instance status
 */
func (ti *TaskInstance) GetStatus() TaskStatus {
	return TaskStatus(ti.Status)
}

/**
 * Set instance status (without updating storage)
 */
func (ti *TaskInstance) SetStatus(status TaskStatus) {
	now := time.Now().Local()
	ti.Status = string(status)
	ti.UpdateTime = &now
	ti.phase = status.Phase()

	if status == TaskStatusInit {
		ti.StartTime = &now
	} else if status == TaskStatusRunning {
		ti.RunningTime = &now
	} else if status.IsFinished() {
		ti.EndTime = &now
	}
}

/**
 * Update end-of-life logs
 */
func (ti *TaskInstance) SetEndLog(endLog string) {
	ti.EndLog = endLog
}

/**
 * Compile YAML file for job creation
 */
func (ti *TaskInstance) Compile() (string, error) {
	if ti.template.Schema == "" {
		return "", nil
	}
	tt := template.New("test").Funcs(template.FuncMap{
		"replaceNewline": replaceNewline,
		"yamlQuote":      yamlQuote,
		"yamlValue":      yamlValue,
		"hasKey":         hasKey,
	})
	// Create template
	tpl, err := tt.Parse(ti.template.Schema)
	if err != nil {
		return "", fmt.Errorf("error in parse template.schema: %v", err)
	}
	args, err := ParseArgs(ti.Args)
	if err != nil {
		return "", fmt.Errorf("error in parseArgs task_obj.value: %v", err)
	}
	extra, err := ti.GetExtra()
	if err != nil {
		return "", fmt.Errorf("error in parseArgs task_obj.extra: %v", err)
	}
	args["_task"] = ti.TaskRec
	args["_extra"] = extra
	args["_tags"] = ti.GetTags()

	var buf bytes.Buffer
	// Execute template and store result in buffer
	err = tpl.Execute(&buf, args)
	if err != nil {
		return "", err
	}
	// Convert buffer content to string and return
	return buf.String(), nil
}

/**
 * Set tags for task instance
 * @param tags map[string]string Tag key-value pairs
 */
func (ti *TaskInstance) SetTags(tags map[string]string) {
	if tags == nil {
		return
	}
	if ti.tags == nil {
		ti.tags = make(map[string]string)
	}
	for k, v := range tags {
		ti.tags[k] = v
	}
}

/**
 * Get tags for task instance
 */
func (ti *TaskInstance) GetTags() map[string]string {
	return ti.tags
}

/**
 * Callback when task instance finishes
 */
func (ti *TaskInstance) SendCallback(message string) error {
	if ti.Callback == "" {
		return nil
	}
	utils.Infof("Task [%s] has finished running, send notification: %s", ti.Title(), ti.Callback)
	type TaskFinishedCallback struct {
		Name    string `json:"name"`
		Uuid    string `json:"uuid"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	var msg TaskFinishedCallback
	if message == "" {
		message = fmt.Sprintf("Task [%s] has finished running", ti.Title())
	}
	msg.Name = ti.Name
	msg.Message = message
	msg.Uuid = ti.UUID
	msg.Status = string(ti.GetStatus())

	data, err := json.Marshal(&msg)
	if err != nil {
		return err
	}
	ss := utils.NewSession(ti.Callback)
	_, err = ss.Post("", data)
	return err
}

/**
 * Get task timeout settings
 */
func (ti *TaskInstance) GetTimeout() Timeout {
	if ti.Timeout == "" {
		return defaultTimeout
	}
	timeout := defaultTimeout
	interval := TimeoutSetting{}
	if err := json.Unmarshal([]byte(ti.Timeout), &interval); err != nil {
		utils.Errorf("Task [%s] Unmarshal 'timeout' failed: %v", ti.Title(), err)
		return timeout
	}
	if interval.Queue != 0 {
		timeout.Queue = time.Duration(interval.Queue) * time.Minute
	}
	if interval.Init != 0 {
		timeout.Init = time.Duration(interval.Init) * time.Minute
	}
	if interval.Running != 0 {
		timeout.Running = time.Duration(interval.Running) * time.Minute
	}
	if interval.Whole != 0 {
		timeout.Whole = time.Duration(interval.Whole) * time.Minute
	}
	return timeout
}

/**
 * Get task instance extra parameters
 * Template can define default extra values, which can be
 * overridden by same-named fields in TaskRec.extra
 */
func (ti *TaskInstance) GetExtra() (map[string]any, error) {
	var tplExtra map[string]any
	var objExtra map[string]any
	var err error
	if ti.template.Extra != "" {
		tplExtra, err = ParseArgs(ti.template.Extra)
		if err != nil {
			return nil, err
		}
	} else {
		tplExtra = make(map[string]any)
	}
	if ti.Extra != "" {
		objExtra, err = ParseArgs(ti.Extra)
		if err != nil {
			return nil, err
		}
	} else {
		objExtra = make(map[string]any)
	}
	for k, v := range objExtra {
		tplExtra[k] = v
	}
	return tplExtra, nil
}

/**
 * Get resource quotas defined for submitted task
 */
func (ti *TaskInstance) GetQuotas() []dao.Quota {
	if ti.Quotas == "" {
		return []dao.Quota{}
	}
	var quotas []dao.Quota
	if err := json.Unmarshal([]byte(ti.Quotas), &quotas); err != nil {
		utils.Errorf("Task [%s] unmarshal 'quotas' failed: %v", ti.Title(), err)
		return []dao.Quota{}
	}
	return quotas
}

/**
 * Allocate resource quotas for task instance
 */
func (ti *TaskInstance) AllocQuotas() error {
	quotas := ti.GetQuotas()
	if err := ti.pool.AllocQuotas(quotas); err != nil {
		return err
	}
	ti.quotas = quotas
	return nil
}

/**
 * Free resource quotas held by task instance
 */
func (ti *TaskInstance) FreeQuotas() {
	if len(ti.quotas) == 0 {
		return
	}
	if err := ti.pool.FreeQuotas(ti.quotas); err != nil {
		utils.Errorf("Task [%s] free quotas failed: %v", ti.Title(), err)
	}
	ti.quotas = []dao.Quota{}
}

/**
 * Custom function: replace newlines with \n
 */
func replaceNewline(i any, indent int) any {
	if s, ok := i.(string); ok {
		indentStr := ""
		for i := 0; i < indent; i++ {
			indentStr += " "
		}
		if strings.Contains(s, "\n") {
			return strings.ReplaceAll("|\n"+s, "\n", "\n"+indentStr)
		}
	}
	return i
}

/**
 * Custom function: check if map contains specified key
 */
func hasKey(i map[string]any, key string) bool {
	_, ok := i[key]
	return ok
}

/**
 * text/template custom function: escape string for YAML format
 */
func yamlQuote(v any) any {
	if v == nil {
		return "\"\""
	}
	return fmt.Sprintf("%q", fmt.Sprintf("%v", v))
}

/**
 * text/template custom function: output v, or defV if v is nil
 */
func yamlValue(v any, defV any) any {
	if v == nil {
		return defV
	} else {
		return v
	}
}

/**
 * Parse JSON values
 */
func ParseArgs(args string) (map[string]any, error) {
	var params map[string]any
	if args == "" {
		return make(map[string]any), nil
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return nil, err
	}
	return params, nil
}

/**
 * Get integer parameter value by name
 */
func GetArgInt(args map[string]any, key string, defVal int) int {
	val, ok := args[key]
	if !ok {
		return defVal
	}
	valInt, ok := val.(int)
	if !ok {
		return defVal
	}
	return valInt
}

/**
 * Get string parameter value by name
 */
func GetArgString(args map[string]any, key string, defVal string) string {
	val, ok := args[key]
	if !ok {
		return defVal
	}
	valStr, ok := val.(string)
	if !ok {
		return defVal
	}
	return valStr
}

/**
 * Get key-value pairs
 */
func GetArgKvs(args map[string]any, name string) map[string]string {
	kvs := make(map[string]string)
	if args == nil {
		return kvs
	}
	v, ok := args[name]
	if !ok {
		return kvs
	}
	arg, ok := v.(map[string]string)
	if !ok {
		utils.Errorf("Argument [%s] isn't map[string]string", name)
	}
	return arg
}
