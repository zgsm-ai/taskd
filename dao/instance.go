package dao

import (
	"fmt"
	"strings"
	"taskd/internal/utils"
	"time"
)

/**
 * Task request submitted by user
 */
type TaskObjRec struct {
	UUID      string `json:"uuid,omitempty"`       // Task UUID
	Parent    string `json:"parent,omitempty"`     // The batch specified by this ID
	Namespace string `json:"namespace,omitempty"`  // Namespace
	Name      string `json:"name,omitempty"`       // Task name
	Project   string `json:"project,omitempty"`    // Project name
	Template  string `json:"template,omitempty"`   // Template name
	Pool      string `json:"pool,omitempty"`       // Task pool
	Extra     string `json:"extra,omitempty"`      // Extra info for template (JSON)
	Args      string `json:"args,omitempty"`       // User arguments for task (JSON)
	Timeout   string `json:"timeout,omitempty"`    // Timeout settings (JSON)
	Quotas    string `json:"quotas,omitempty"`     // Resource quotas (JSON)
	Tags      string `json:"tags,omitempty"`       // Tags affecting scheduling (JSON key=value)
	Callback  string `json:"callback,omitempty"`   // Callback URL
	CreatedBy string `json:"created_by,omitempty"` // Creator
}

/**
 * Task runtime records
 */
type TaskRuntimeRec struct {
	YamlContent string     `json:"yaml_content,omitempty"` // Deployment file content
	CreateTime  *time.Time `json:"create_time"`            // Creation time
	StartTime   *time.Time `json:"start_time"`             // Start time
	RunningTime *time.Time `json:"running_time"`           // Running start time
	EndTime     *time.Time `json:"end_time"`               // End time
	UpdateTime  *time.Time `json:"update_time"`            // Last update time
	Status      string     `json:"status"`                 // Task status
	Error       string     `json:"error"`                  // Error message
	Warning     string     `json:"warning"`                // Warning message
	EndLog      string     `json:"end_log"`                // Final logs
}

/**
 * Resource quota
 */
type Quota struct {
	ResName string `json:"res_name,omitempty"` // Resource name
	ResNum  int64  `json:"res_num,omitempty"`  // Resource quantity
	ResFmt  string `json:"res_fmt,omitempty"`  // Resource unit
}

/**
 * Task instance
 */
type TaskRec struct {
	TaskObjRec
	TaskRuntimeRec
}

/**
 * Parameters for listing tasks
 */
type ListTasksArgs struct {
	Uuid      string `form:"uuid"`      // UUID
	Namespace string `form:"namespace"` // Namespace
	Name      string `form:"name"`      // Task name
	Template  string `form:"template"`  // Template name
	Project   string `form:"project"`   // Project name
	Pool      string `form:"pool"`      // Target resource pool
	Owner     string `form:"owner"`     // Task owner
	Status    string `form:"status"`    // Task status
	Page      int    `form:"page"`      // Page number
	PageSize  int    `form:"pageSize"`  // Items per page
	Sort      string `form:"sort"`      // Sort field
	Verbose   bool   `form:"verbose"`   // Output task details
}

/**
 * List tasks result
 */
type ListTasksResult struct {
	Total int       `json:"total"`
	List  []TaskRec `json:"list"`
}

//
// Redis index structure:
//
// tasks:---+--objects:---+
//                        +--<UUID> -> {}
//
//          +--indexes:--+
//                        +---name:----+
//                                     +--<name>:----+
//                                                   +-<UUID> -> <name>
//
//                        +---status:--+
//                                     +--<status>:--+
//                                                   +-<UUID> -> <status>
//
//          +--running:---+
//                        +---<UUID> -> <UUID>
//
// Complete task data is stored under `tasks:objects:<UUID>`
// When task initializes:
//   1. Create indexes for namespace,name,project,template,pool,created_by under `tasks:indexes:`
//      e.g. tasks:indexes:name:<name>:<UUID> means a task with name <name> and UUID <UUID>
//   2. Running task UUIDs are stored in `tasks:running:<UUID>` and removed when finished
// When task finishes:
//   1. Delete `tasks:running:<UUID>` key-value
//   2. Create status index for task under `tasks:indexes:`
//

/**
 * Get task object key in Redis
 */
func objKey(uuid string) string {
	return fmt.Sprintf("tasks:objects:%s", uuid)
}

/**
 * Get task object key stored in Redis
 */
func (ti *TaskRec) objKey() string {
	return fmt.Sprintf("tasks:objects:%s", ti.UUID)
}

/**
 * Create record
 */
func (ti *TaskRec) Create() error {
	if err := SetJSON(ti.objKey(), ti, 365*24*time.Hour); err != nil {
		return err
	}
	// 1. Index by task name
	nameKey := fmt.Sprintf("tasks:indexes:name:%s:%s", ti.Name, ti.UUID)
	if err := SetJSON(nameKey, ti.Name, 365*24*time.Hour); err != nil {
		return err
	}

	// 2. Index by namespace
	nsKey := fmt.Sprintf("tasks:indexes:namespace:%s:%s", ti.Namespace, ti.UUID)
	if err := SetJSON(nsKey, ti.Namespace, 365*24*time.Hour); err != nil {
		return err
	}

	// 3. Index by project
	projectKey := fmt.Sprintf("tasks:indexes:project:%s:%s", ti.Project, ti.UUID)
	if err := SetJSON(projectKey, ti.Project, 365*24*time.Hour); err != nil {
		return err
	}

	// 4. Index by template
	templateKey := fmt.Sprintf("tasks:indexes:template:%s:%s", ti.Template, ti.UUID)
	if err := SetJSON(templateKey, ti.Template, 365*24*time.Hour); err != nil {
		return err
	}

	// 5. Index by pool
	poolKey := fmt.Sprintf("tasks:indexes:pool:%s:%s", ti.Pool, ti.UUID)
	if err := SetJSON(poolKey, ti.Pool, 365*24*time.Hour); err != nil {
		return err
	}

	// 6. Index by creator
	creatorKey := fmt.Sprintf("tasks:indexes:created_by:%s:%s", ti.CreatedBy, ti.UUID)
	if err := SetJSON(creatorKey, ti.CreatedBy, 365*24*time.Hour); err != nil {
		return err
	}

	runningKey := fmt.Sprintf("tasks:running:%s", ti.UUID)
	if err := SetJSON(runningKey, ti.UUID, 365*24*time.Hour); err != nil {
		return err
	}
	return nil
}

/**
 * Update record
 */
func (ti *TaskRec) Update() error {
	return SetJSON(ti.objKey(), ti, 365*24*time.Hour)
}

/**
 * Delete record
 */
func (ti *TaskRec) Delete() {
	Del(ti.objKey())
}

/**
 * Save task "corpse" after completion and build indexes
 * Move task from running list (tasks:running) to objects list (tasks:objects)
 * and build indexes for the task
 */
func (ti *TaskRec) Bury() error {
	// 1. Remove from running list
	runningKey := fmt.Sprintf("tasks:running:%s", ti.UUID)
	Del(runningKey)

	// 2. Update object data
	if err := ti.Update(); err != nil {
		return err
	}
	// 3. Index by status
	statusKey := fmt.Sprintf("tasks:indexes:status:%s:%s", ti.Status, ti.UUID)
	if err := SetJSON(statusKey, ti.Status, 365*24*time.Hour); err != nil {
		return err
	}

	return nil
}

/**
 * Extract UUIDs from keys
 * Key format example: tasks:indexes:name:c1:39c28647-d0d1-40ec-9902-d73a375e0fab
 * Need to get the last part as UUID
 */
func getUUIDs(keys []string) []string {
	uuids := make([]string, 0, len(keys))
	for _, key := range keys {
		parts := strings.Split(key, ":")
		if len(parts) > 0 {
			uuids = append(uuids, parts[len(parts)-1])
		}
	}
	return uuids
}

/**
 * Get task objects for given UUIDs
 */
func getTasks(uuids []string) []TaskRec {
	var tasks []TaskRec
	for _, uuid := range uuids {
		var task TaskRec
		if err := GetJSON(objKey(uuid), &task); err != nil {
			utils.Errorf("Failed to get task %s: %s", uuid, err.Error())
			continue
		}
		tasks = append(tasks, task)
	}
	return tasks
}

/**
 * Return intersection of two string arrays
 */
func intersectionKeys(lhs, rhs []string) []string {
	// Create map to store elements from lhs
	lhsMap := make(map[string]bool)
	for _, key := range lhs {
		lhsMap[key] = true
	}

	// Create slice to store intersection
	var intersection []string
	for _, key := range rhs {
		if lhsMap[key] {
			// If an element in rhs exists in lhsMap, it's part of the intersection
			intersection = append(intersection, key)
			// Remove from lhsMap to prevent duplicates
			delete(lhsMap, key)
		}
	}

	return intersection
}

type matcher struct {
	Error     error
	Activated bool
	Result    []string
}

/**
 * Match new condition using intersection approach
 */
func (m *matcher) matchCondition(prefix, cond string) error {
	if cond == "" {
		return nil
	}
	keys, err := KeysByPrefix(fmt.Sprintf("%s:%s:*", prefix, cond))
	if err != nil {
		m.Error = err
		return err
	}
	keys = getUUIDs(keys)
	if !m.Activated {
		m.Result = keys
		m.Activated = true
	} else {
		m.Result = intersectionKeys(m.Result, keys)
	}
	return nil
}

/**
 * List tasks
 * When task finishes, move to Redis completed KV list as status won't change
 * Create indexes based on possible future query patterns
 */
func ListTasks(args *ListTasksArgs) (ListTasksResult, error) {
	var taskResult ListTasksResult
	var keys []string
	var err error
	var m matcher

	m.matchCondition("tasks:indexes:name", args.Name)
	m.matchCondition("tasks:indexes:template", args.Template)
	m.matchCondition("tasks:indexes:project", args.Project)
	m.matchCondition("tasks:indexes:pool", args.Pool)
	m.matchCondition("tasks:indexes:namespace", args.Namespace)
	m.matchCondition("tasks:indexes:created_by", args.Owner)
	m.matchCondition("tasks:indexes:status", args.Status)

	if m.Error != nil {
		return taskResult, m.Error
	}
	if m.Activated {
		keys = m.Result
	} else {
		// If no query conditions, get all finished tasks
		keys, err = KeysByPrefix("tasks:objects:*")
		if err != nil {
			return taskResult, err
		}
		keys = getUUIDs(keys)
	}
	// Set total count
	taskResult.Total = len(keys)

	// Handle pagination
	if args.Page > 0 && args.PageSize > 0 {
		start := (args.Page - 1) * args.PageSize
		end := start + args.PageSize
		if end > len(keys) {
			end = len(keys)
		}
		keys = keys[start:end]
	}
	taskResult.List = getTasks(keys)
	if !args.Verbose {
		for i := range taskResult.List {
			taskResult.List[i].Extra = ""
			taskResult.List[i].Args = ""
			taskResult.List[i].YamlContent = ""
		}
	}
	return taskResult, nil
}

/**
 * Load all unfinished tasks from database
 */
func LoadTasks_NotFinished() ([]TaskRec, error) {
	var tasks []TaskRec

	keys, err := KeysByPrefix("tasks:running:")
	if err != nil {
		return tasks, err
	}
	keys = getUUIDs(keys)
	tasks = getTasks(keys)
	return tasks, nil
}

/**
 * Load task object from database
 */
func LoadTask(uuid string) (*TaskRec, error) {
	var ti TaskRec
	if err := GetJSON(objKey(uuid), &ti); err != nil {
		return nil, err
	}
	return &ti, nil
}

/**
 * Check if task with given UUID exists
 */
func ExistTask(uuid string) (bool, error) {
	return Exists(objKey(uuid))
}
