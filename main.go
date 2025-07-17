package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"taskd/controllers"
	"taskd/dao"
	_ "taskd/docs"
	"taskd/internal/custom"
	"taskd/internal/flow"
	"taskd/internal/task"
	"taskd/internal/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Task Management
// @version 1.0
// @description Task Management System
// @termsOfService http://www.sangfor.com.cn
// @contact.name Zhaojin Zhang,Bochun Zheng
// @contact.url http://www.sangfor.com.cn
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /taskd/api
// @query.collection.format multi
/*
 * Main entry point, initializes configuration and starts HTTP server
 */
func main() {
	var c = &utils.Config{}

	printVersions()
	// Initialize configuration file
	if err := c.Init("./env.yaml"); err != nil {
		panic(fmt.Errorf("Init('./env.yaml') failed: %v", err))
	}
	initLogger(&c.Logger)
	if err := dao.InitDB(c.Db); err != nil {
		panic(fmt.Errorf("InitDB failed:%v", err))
	}
	if err := dao.InitRedis(c.Redis.Addr, c.Redis.Password, c.Redis.DB); err != nil {
		panic(fmt.Errorf("InitRedis failed: %v", err))
	}
	utils.SetProxyUrl(c.WeChat.Enable, c.WeChat.Proxy, c.WeChat.RobotURL)
	utils.InitLokiLog(c.LokiURL)

	initProcess(c)

	runHttpServer(&c.Server)
}

var SoftwareVer = ""
var BuildTime = ""
var BuildTag = ""
var BuildCommitId = ""

/*
 * Print software version information
 */
func printVersions() {
	fmt.Printf("Version %s\n", SoftwareVer)
	fmt.Printf("Build Time: %s\n", BuildTime)
	fmt.Printf("Build Tag: %s\n", BuildTag)
	fmt.Printf("Build Commit ID: %s\n", BuildCommitId)
}

/*
 * Initialize logger configuration
 */
func initLogger(c *utils.LoggerConfig) {
	if c.Format == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else if c.Format == "text" {
		logrus.SetFormatter(&logrus.TextFormatter{})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{})
	}
	if c.Output == "stdout" {
		logrus.SetOutput(os.Stdout)
	} else if c.Output == "stderr" {
		logrus.SetOutput(os.Stderr)
	} else {
		logrus.SetOutput(os.Stdout)
	}
	level, err := logrus.ParseLevel(c.Level)
	if err != nil {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(level)
	}
}

// Logger is a custom middleware function
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record start time
		start := time.Now()

		// Process request
		c.Next()

		// Record end time
		end := time.Now()

		// Calculate request processing time
		latency := end.Sub(start)

		// Get request info
		requestMethod := c.Request.Method
		requestURL := c.Request.URL.String()
		if strings.Contains(requestURL, "logs") {
			return
		}

		statusCode := c.Writer.Status()

		// Log detailed request/response info
		fmt.Printf("[GIN] %v | %3d | %13v | %15s | %-7s %s\n",
			end.Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			c.ClientIP(),
			requestMethod,
			requestURL,
		)
	}
}

/*
 * Start HTTP server and register routes
 */
func runHttpServer(c *utils.ServerConfig) {
	if !c.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	if !c.Debug && c.Logger {
		r.Use(Logger())
	}

	// Register Swagger handler
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	apiv1 := r.Group("/taskd/api/v1")
	{
		// Tasks
		apiv1.POST("/tasks", controllers.TaskCommit)
		apiv1.GET("/tasks", controllers.ListTasks)
		apiv1.GET("/tasks/:uuid", controllers.TaskData)
		apiv1.GET("/tasks/:uuid/status", controllers.TaskStatus)
		apiv1.GET("/tasks/:uuid/logs", controllers.TaskLogs)
		apiv1.GET("/tasks/:uuid/tags", controllers.TaskGetTags)
		apiv1.POST("/tasks/:uuid/tags", controllers.TaskTags)
		apiv1.DELETE("/tasks/:uuid", controllers.TaskStop)

		// Task templates
		apiv1.POST("/templates", controllers.AddTemplate)
		apiv1.PUT("/templates/:name", controllers.UpdateTemplate)
		apiv1.GET("/templates", controllers.ListTemplates)
		apiv1.GET("/templates/:name", controllers.GetTemplate)
		apiv1.DELETE("/templates/:name", controllers.DeleteTemplate)

		// Task pools
		apiv1.POST("/pools", controllers.AddPool)
		apiv1.GET("/pools", controllers.ListPools)
		apiv1.GET("/pools/:name", controllers.GetPool)
		apiv1.PUT("/pools/:name", controllers.UpdatePool)
		apiv1.DELETE("/pools/:name", controllers.DeletePool)
	}
	err := r.Run(c.ListenAddr)
	if err != nil {
		log.Fatal(err)
	}
}

/*
 * Initialize task management submodules
 * @param c Configuration object
 */
func initProcess(c *utils.Config) {
	defer func() {
		if r := recover(); r != nil {
			// Handle errors
			utils.Errorf("initProcess panic: %v", r)
			utils.ReportAlerts()
		}
	}()
	task.SetDefaultTimeout(task.TimeoutSetting{
		Queue:   c.Timeout.PhaseQueueDefault,
		Init:    c.Timeout.PhaseInitDefault,
		Running: c.Timeout.PhaseRunningDefault,
		Whole:   c.Timeout.PhaseWholeDefault,
	})
	// Register task engines
	task.RegisterEngine(task.PodEngine, custom.NewPod, custom.InitK8sExtension, flow.NewPoller)
	task.RegisterEngine(task.CrdEngine, custom.NewCrd, custom.InitK8sExtension, flow.NewPoller)
	task.RegisterEngine(task.KFJobEngine, custom.NewKFJob, custom.InitK8sExtension, flow.NewPoller)
	task.RegisterEngine(task.RpcEngine, custom.NewRpc, nil, flow.NewReactor)

	if err := flow.Init(); err != nil {
		panic(err)
	}
	// Load historical tasks from cache
	if err := flow.ReloadHistoryTasks(); err != nil {
		panic(fmt.Errorf("load history tasks failed: %v", err))
	}
}
