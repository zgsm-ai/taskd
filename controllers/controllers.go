package controllers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"taskd/dao"
	"taskd/internal/flow"
	"taskd/internal/utils"
	"taskd/service"

	"github.com/gin-gonic/gin"
)

/**
 * API response structure
 */
type ResponseData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
}

/**
 * Normal API response
 */
func respOK(c *gin.Context, data any) {
	utils.Debugf("request: %+v, response: %+v", c.Request.RequestURI, data)
	c.JSON(http.StatusOK, data)
}

/**
 * Error API response
 */
func respError(c *gin.Context, code int, err error) {
	utils.Errorf("request: %+v, error: %s", c.Request.RequestURI, err.Error())
	if httpErr, ok := err.(*utils.HttpError); ok {
		c.JSON(httpErr.Code(), ResponseData{
			Code:    strconv.Itoa(httpErr.Code()),
			Message: httpErr.Error(),
			Success: false,
			Data:    nil,
		})
	} else {
		c.JSON(code, ResponseData{
			Code:    strconv.Itoa(code),
			Message: err.Error(),
			Success: false,
			Data:    nil,
		})
	}
}

/**
 * Streaming response
 */
func respStream(c *gin.Context, f io.ReadCloser) {
	// Set response headers
	c.Header("Content-Type", "text/plain")

	defer f.Close()

	// Read and send log output line by line
	buf := make([]byte, 1024)
	for {
		n, err := f.Read(buf)
		if err != nil {
			if err != io.EOF {
				c.String(500, err.Error())
			}
			return
		}

		// Send log output to client
		if n > 0 {
			if _, err := c.Writer.Write(buf[:n]); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}

/**
 * Template creation result
 */
type AddTemplateResult struct {
	Name string `json:"name"`
}

// ListTemplates
// @Summary List task templates
// @Schemes
// @Description Task templates
// @Tags TaskTemplates
// @Param verbose query bool false "Include details"
// @Accept json
// @Produce json
// @Success 200 {array} dao.TemplateRec "List of task template names"
// @Failure 400 {object} ResponseData "Invalid request"
// @Failure 500 {object} ResponseData "Internal server error"
// @Router /v1/templates [GET]
func ListTemplates(c *gin.Context) {
	verbose := strings.EqualFold(c.Query("verbose"), "true")
	names, err := dao.ListTemplates(verbose)
	if err != nil {
		respError(c, http.StatusInternalServerError, err)
		return
	}
	respOK(c, names)
}

// GetTemplate
// @Summary Get task template
// @Schemes
// @Description Get task template
// @Tags TaskTemplates
// @Param name path string true "Template name"
// @Accept json
// @Produce json
// @Success 200 {object} dao.TemplateRec "Task template details"
// @Failure 400 {object} ResponseData "Invalid request"
// @Failure 404 {object} ResponseData "Template not found"
// @Failure 500 {object} ResponseData "Internal server error"
// @Router /v1/templates/{name} [GET]
func GetTemplate(c *gin.Context) {
	td, err := dao.LoadTemplate(c.Param("name"))
	if err != nil {
		respError(c, http.StatusInternalServerError, err)
		return
	}
	respOK(c, td)
}

// AddTemplate
// @Summary Create task template
// @Schemes
// @Description Create task template
// @Tags TaskTemplates
// @Param templates body dao.TemplateRec true "Template definition"
// @Accept json
// @Produce json
// @Success 200 {object} AddTemplateResult "Created template (ID+Name)"
// @Router /v1/templates [POST]
func AddTemplate(c *gin.Context) {
	var req dao.TemplateRec
	if err := c.ShouldBindJSON(&req); err != nil {
		respError(c, http.StatusBadRequest, err)
		return
	}

	if req.Name == "" {
		respError(c, http.StatusBadRequest, fmt.Errorf("template name cannot be empty"))
		return
	}

	if len(req.Name) > 64 {
		respError(c, http.StatusBadRequest, fmt.Errorf("template name length cannot exceed 64 characters"))
		return
	}

	if err := service.AddTemplate(&req); err != nil {
		respError(c, http.StatusInternalServerError, err)
		return
	}
	respOK(c, AddTemplateResult{
		Name: req.Name,
	})
}

// DeleteTemplate
// @Summary Delete task template
// @Schemes
// @Description Delete task template
// @Tags TaskTemplates
// @Param name path string true "Template name"
// @Accept json
// @Produce json
// @Success 200 {string} string "Delete success message"
// @Failure 400 {object} ResponseData "Tasks using this template exist"
// @Failure 404 {object} ResponseData "Template not found"
// @Failure 500 {object} ResponseData "Internal server error"
// @Router /v1/templates/{name} [DELETE]
func DeleteTemplate(c *gin.Context) {
	name := c.Param("name")
	if err := service.DeleteTemplate(name); err != nil {
		respError(c, http.StatusInternalServerError, err)
		return
	}
	respOK(c, fmt.Sprintf("template [%s] deleted", name))
}

// UpdateTemplate
// @Summary Update task template
// @Schemes
// @Description Update task template
// @Tags TaskTemplates
// @Param name path string true "Template name"
// @Param templates body dao.TemplateRec true "Template definition"
// @Accept json
// @Produce json
// @Success 200 {object} AddTemplateResult "Template creation result (ID+Name)"
// @Failure 400 {object} ResponseData "Invalid request"
// @Failure 500 {object} ResponseData "Internal server error"
// @Router /v1/templates/{name} [PUT]
func UpdateTemplate(c *gin.Context) {
	name := c.Param("name")
	var req dao.TemplateRec
	if err := c.ShouldBindJSON(&req); err != nil {
		respError(c, http.StatusBadRequest, err)
		return
	}
	if req.Name == "" {
		req.Name = name
	}
	if name != req.Name {
		respError(c, http.StatusBadRequest, fmt.Errorf("template name modification is not allowed"))
		return
	}
	if err := service.UpdateTemplate(&req); err != nil {
		respError(c, http.StatusInternalServerError, err)
		return
	}
	respOK(c, AddTemplateResult{
		Name: req.Name,
	})
}

// ListPools
// @Summary List task pools
// @Schemes
// @Description List task pool information including resource usage and running task overview
// @Tags TaskPools
// @Accept json
// @Produce json
// @Success 200 {array} task.TaskPoolSummary "Queue information"
// @Router /v1/pools [GET]
func ListPools(c *gin.Context) {
	pools := flow.ListPools()
	respOK(c, pools)
}

// GetPool
// @Summary Get task pool details
// @Schemes
// @Description Get task pool details
// @Tags TaskPools
// @Param name path string true "Pool name"
// @Param verbose query bool false "Get detailed info"
// @Accept json
// @Produce json
// @Success 200 {object} task.TaskPoolDetail "Pool information"
// @Router /v1/pools/{name} [GET]
func GetPool(c *gin.Context) {
	name := c.Param("name")
	verbose := strings.EqualFold(c.Query("verbose"), "true")

	pool := flow.GetPool(name)
	if pool == nil {
		respError(c, http.StatusBadRequest, fmt.Errorf("pool [%s] is not exist", name))
		return
	}
	if !verbose {
		respOK(c, pool.GetSummary())
		return
	}
	respOK(c, pool.GetDetail())
}

// AddPool
// @Summary Add a task pool
// @Schemes
// @Description Add a task pool
// @Tags TaskPools
// @Param pool body service.TaskPoolArgs true "Task pool"
// @Accept json
// @Produce json
// @Success 200 {object} service.TaskPoolResult "Task pool result"
// @Router /v1/pools [POST]
func AddPool(c *gin.Context) {
	var req service.TaskPoolArgs
	if err := c.ShouldBindJSON(&req); err != nil {
		respError(c, http.StatusBadRequest, err)
		return
	}
	if err := service.AddPool(&req); err != nil {
		respError(c, http.StatusInternalServerError, err)
		return
	}
	respOK(c, service.TaskPoolResult{
		PoolId: req.PoolId,
	})
}

// DeletePool
// @Summary Delete task pool
// @Schemes
// @Description Delete task pool and associated PoolResource
// @Tags TaskPools
// @Param name path string true "Pool name"
// @Accept json
// @Produce json
// @Success 200 {string} string "Delete success message"
// @Failure 400 {object} ResponseData "Running tasks exist"
// @Failure 404 {object} ResponseData "Pool not found"
// @Failure 500 {object} ResponseData "Internal server error"
// @Router /v1/pools/{name} [DELETE]
func DeletePool(c *gin.Context) {
	poolId := c.Param("name")
	if err := service.DeletePool(poolId); err != nil {
		respError(c, http.StatusInternalServerError, err)
		return
	}
	respOK(c, fmt.Sprintf("pool [%s] deleted", poolId))
}

// UpdatePool
// @Summary Update task pool
// @Schemes
// @Description Update task pool definition
// @Tags TaskPools
// @Param name path string true "Task pool ID"
// @Param pools body service.TaskPoolArgs true "Task pool"
// @Accept json
// @Produce json
// @Success 200 {object} service.TaskPoolResult "Update task pool"
// @Router /v1/pools/{name} [PUT]
func UpdatePool(c *gin.Context) {
	name := c.Param("name")
	var req service.TaskPoolArgs
	if err := c.ShouldBindJSON(&req); err != nil {
		respError(c, http.StatusBadRequest, err)
		return
	}
	if req.PoolId == "" {
		req.PoolId = name
	}
	if name != req.PoolId {
		respError(c, http.StatusBadRequest, fmt.Errorf("pool name modification is not allowed"))
		return
	}
	if err := service.UpdatePool(&req); err != nil {
		respError(c, http.StatusInternalServerError, err)
		return
	}
	respOK(c, service.TaskPoolResult{
		PoolId: req.PoolId,
	})
}

// ListTasks
// @Summary List tasks
// @Schemes
// @Description List tasks
// @Tags Tasks
// @Param req query dao.ListTasksArgs true "Query parameters"
// @Accept json
// @Produce json
// @Success 200 {object} dao.ListTasksResult "Task list result"
// @Router /v1/tasks [GET]
func ListTasks(c *gin.Context) {
	var args dao.ListTasksArgs
	if err := c.ShouldBindQuery(&args); err != nil {
		respError(c, http.StatusBadRequest, err)
		return
	}
	result, err := dao.ListTasks(&args)
	if err != nil {
		respError(c, http.StatusInternalServerError, err)
		return
	}
	respOK(c, result)
}

// TaskCommit
// @Summary Submit task
// @Schemes
// @Description Submit task
// @Tags Tasks
// @Param task body dao.TaskObjRec true "Task object"
// @Accept json
// @Produce json
// @Success 200 {object} service.TaskCommitResult "Task commit result (UUID, RunID)"
// @Router /v1/tasks [POST]
func TaskCommit(c *gin.Context) {
	var req dao.TaskObjRec
	if err := c.ShouldBindJSON(&req); err != nil {
		respError(c, http.StatusBadRequest, err)
		return
	}

	if req.Template == "" {
		respError(c, http.StatusBadRequest, fmt.Errorf("task template cannot be empty"))
		return
	}

	result, err := service.TaskCommit(&req)
	if err != nil {
		respError(c, http.StatusInternalServerError, err)
		return
	}
	respOK(c, result)
}

// TaskStop
// @Summary Stop task
// @Schemes
// @Description Stop task
// @Tags Tasks
// @Param uuid path string true "Task UUID"
// @Accept json
// @Produce json
// @Success 200 {string} string "Operation success message"
// @Router /v1/tasks/{uuid} [DELETE]
func TaskStop(c *gin.Context) {
	if err := service.TaskStop(c.Param("uuid")); err != nil {
		respError(c, http.StatusInternalServerError, err)
		return
	}

	respOK(c, "task stopped")
}

// TaskData
// @Summary Get task metadata
// @Schemes
// @Description Get task metadata
// @Tags Tasks
// @Param uuid path string true "Task UUID"
// @Accept json
// @Produce json
// @Success 200 {object} dao.TaskRec "Task object details"
// @Router /v1/tasks/{uuid} [GET]
func TaskData(c *gin.Context) {
	to, err := service.GetTask(c.Param("uuid"))
	if err != nil {
		respError(c, http.StatusBadRequest, err)
		return
	}

	respOK(c, to)
}

// TaskStatus
// @Summary Get task status
// @Schemes
// @Description Get task status
// @Tags Tasks
// @Param uuid path string true "Task UUID"
// @Accept json
// @Produce json
// @Success 200 {object} service.TaskStatusResult "Task status information"
// @Router /v1/tasks/{uuid}/status [GET]
func TaskStatus(c *gin.Context) {
	result, err := service.TaskStatus(c.Param("uuid"))
	if err != nil {
		respError(c, http.StatusBadRequest, err)
		return
	}
	respOK(c, result)
}

// TaskLogs
// @Summary Get task logs
// @Description Get task logs with stream support (tail/follow) and regular pagination
// @Tags Tasks
// @Param uuid path string true "Task UUID" required
// @Param req query service.TaskLogsArgs false "Log parameters"
// @Accept json
// @Produce json, plain/text
// @Success 200 {object} service.TaskLogsResult "Log results"
// @Success 200 {string} string "Streaming log output"
// @Failure 400 {object} ResponseData "Invalid request"
// @Failure 404 {object} ResponseData "Task not found"
// @Failure 500 {object} ResponseData "Internal server error"
// @Router /v1/tasks/{uuid}/logs [GET]
func TaskLogs(c *gin.Context) {
	var args service.TaskLogsArgs
	if err := c.ShouldBindQuery(&args); err != nil {
		respError(c, http.StatusBadRequest, err)
		return
	}
	// Limit maximum query items to 1000
	if args.Tail > 1000 {
		args.Tail = 1000
	}
	if args.Follow {
		f, err := service.TaskFollowLogs(c.Param("uuid"), &args)
		if err != nil {
			respError(c, http.StatusInternalServerError, err)
			return
		}
		respStream(c, f)
		return
	}
	result, err := service.TaskLogs(c.Param("uuid"), &args)
	if err != nil {
		respError(c, http.StatusInternalServerError, err)
		return
	}
	respOK(c, result)
}

// TaskTags
// @Summary Tag task
// @Schemes
// @Description Tag task, usually to notify scheduler for specific handling strategy like guaranteed task or idle task
// @Tags Tasks
// @Param uuid path string true "Task UUID"
// @Param tags body map[string]string false "Tag content in Key=Value format, multiple tags can be set simultaneously"
// @Accept json
// @Produce json
// @Success 200 {object} service.TaskTagsResult "All tags of the task"
// @Router /v1/tasks/{uuid}/tags [POST]
func TaskTags(c *gin.Context) {
	var tags map[string]string
	if err := c.ShouldBindJSON(&tags); err != nil {
		respError(c, http.StatusBadRequest, err)
		return
	}
	newTags, err := service.TaskTags(c.Param("uuid"), tags)
	if err != nil {
		respError(c, http.StatusInternalServerError, err)
		return
	}
	respOK(c, newTags)
}

// TaskGetTags
// @Summary Get task tags
// @Schemes
// @Description Get task tags
// @Tags Tasks
// @Param uuid path string true "Task UUID"
// @Accept json
// @Produce json
// @Success 200 {object} service.TaskTagsResult "Task tags"
// @Router /v1/tasks/{uuid}/tags [GET]
func TaskGetTags(c *gin.Context) {
	result, err := service.GetTaskTags(c.Param("uuid"))
	if err != nil {
		respError(c, http.StatusInternalServerError, err)
		return
	}
	respOK(c, result)
}
