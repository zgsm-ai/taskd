package flow

import (
	"fmt"
	"io"
	"reflect"
	"taskd/dao"
	"taskd/internal/task"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
)

// Assume task is the package name

// Mock task
type mockTaskJob struct {
	status task.TaskStatus
	task.TaskInstance
}

func (mtj *mockTaskJob) FetchStatus() task.TaskStatus {
	return mtj.status
}
func (mtj *mockTaskJob) CustomMetrics() *task.Metric {
	return &task.Metric{}
}
func (mtj *mockTaskJob) FollowLogs(string, bool, int64) (io.ReadCloser, error) {
	return nil, nil
}

func (mtj *mockTaskJob) Logs(string, int64) ([]task.EntityLogs, error) {
	return nil, nil
}

func (mtj *mockTaskJob) Start() error {
	return nil
}

func (mtj *mockTaskJob) Stop() error {
	return nil
}

func (mtj *mockTaskJob) Engine() task.TaskEngineKind {
	return "mock"
}

// Test case 1: Verify behavior when job status is completed, expect sendFinishedChan to be called
func TestHandleRunningJob_CompletedStatus(t *testing.T) {
	Convey("当作业状态已完成时，应该调用 sendFinishedChan", t, func() {
		job := &mockTaskJob{status: task.TaskStatusSucceeded}
		tp := &task.TaskPool{}
		job.AttachPool(tp)
		sendFinishedChanCalled := false
		gomonkey.ApplyMethod(reflect.TypeOf(tp), "SendFinishedChan", func(_ *task.TaskPool, job task.TaskJob) {
			sendFinishedChanCalled = true
		})
		gomonkey.ApplyFunc(dao.SetJSON, func(string, any, time.Duration) error {
			return nil
		})
		dealRunningJob(job)

		So(sendFinishedChanCalled, ShouldBeTrue)
	})
}

func TestInit(t *testing.T) {
	Convey("测试初始化函数", t, func() {
		Convey("正常初始化", func() {
			patches := gomonkey.ApplyFunc(initPools, func() error {
				return nil
			})
			defer patches.Reset()

			err := Init()
			So(err, ShouldBeNil)
			So(allJobs, ShouldNotBeNil)
			So(allPools, ShouldNotBeNil)
		})

		Convey("初始化pool失败", func() {
			patches := gomonkey.ApplyFunc(initPools, func() error {
				return fmt.Errorf("init pools failed")
			})
			defer patches.Reset()

			err := Init()
			So(err, ShouldNotBeNil)
		})
	})
}

func TestReloadPoolConfigs(t *testing.T) {
	Convey("测试重载池配置", t, func() {
		mockPool := &task.TaskPool{}
		mockPool.Pool.PoolId = "test-pool"

		allPools = map[string]*task.TaskPool{
			"test-pool": mockPool,
		}

		Convey("重载存在的池", func() {
			patches := gomonkey.ApplyMethod(reflect.TypeOf(mockPool), "ReloadResources", func(*task.TaskPool) error {
				return nil
			})
			defer patches.Reset()

			err := ReloadPoolConfigs("test-pool")
			So(err, ShouldBeNil)
		})

		Convey("重载不存在的池", func() {
			err := ReloadPoolConfigs("non-existent-pool")
			So(err, ShouldBeNil)
		})

		Convey("重载失败", func() {
			patches := gomonkey.ApplyMethod(reflect.TypeOf(mockPool), "ReloadResources", func(*task.TaskPool) error {
				return fmt.Errorf("reload failed")
			})
			defer patches.Reset()

			err := ReloadPoolConfigs("test-pool")
			So(err, ShouldNotBeNil)
		})
	})
}

func TestReloadHistoryTasks(t *testing.T) {
	Convey("测试重载历史任务", t, func() {
		Convey("没有未完成任务", func() {
			patches := gomonkey.ApplyFunc(dao.LoadTasks_NotFinished, func() ([]dao.TaskRec, error) {
				return []dao.TaskRec{}, nil
			})
			defer patches.Reset()

			err := ReloadHistoryTasks()
			So(err, ShouldBeNil)
		})

		Convey("有未完成任务", func() {
			testTasks := []dao.TaskRec{
				{TaskObjRec: dao.TaskObjRec{UUID: "task1", Template: "template1"}},
				{TaskObjRec: dao.TaskObjRec{UUID: "task2", Template: "template2"}},
			}

			patches := gomonkey.ApplyFunc(dao.LoadTasks_NotFinished, func() ([]dao.TaskRec, error) {
				return testTasks, nil
			})
			patches.ApplyFunc(PoolNewJob, func(tr *dao.TaskRec) (task.TaskJob, error) {
				return &mockTaskJob{}, nil
			})
			defer patches.Reset()

			err := ReloadHistoryTasks()
			So(err, ShouldBeNil)
		})

		Convey("加载任务失败", func() {
			patches := gomonkey.ApplyFunc(dao.LoadTasks_NotFinished, func() ([]dao.TaskRec, error) {
				return nil, fmt.Errorf("load failed")
			})
			defer patches.Reset()

			err := ReloadHistoryTasks()
			So(err, ShouldNotBeNil)
		})
	})
}

func TestListPools(t *testing.T) {
	Convey("测试ListPools函数", t, func() {
		pool1 := &task.TaskPool{}
		pool1.Pool.PoolId = "pool1"
		pool2 := &task.TaskPool{}
		pool2.Pool.PoolId = "pool2"

		allPools = map[string]*task.TaskPool{
			"pool1": pool1,
			"pool2": pool2,
		}

		pools := ListPools()
		So(len(pools), ShouldEqual, 2)
		So(pools[0].PoolId, ShouldEqual, "pool1")
		So(pools[1].PoolId, ShouldEqual, "pool2")
	})
}

func TestGetPool(t *testing.T) {
	Convey("测试GetPool函数", t, func() {
		testPool := &task.TaskPool{}
		testPool.Pool.PoolId = "test-pool"
		allPools = map[string]*task.TaskPool{
			"test-pool": testPool,
		}

		Convey("获取存在的池", func() {
			pool := GetPool("test-pool")
			So(pool, ShouldEqual, testPool)
		})

		Convey("获取不存在的池", func() {
			pool := GetPool("non-existent-pool")
			So(pool, ShouldBeNil)
		})
	})
}

func TestRemovePool(t *testing.T) {
	Convey("测试RemovePool函数", t, func() {
		testPool := &task.TaskPool{}
		testPool.Pool.PoolId = "test-pool"
		allPools = map[string]*task.TaskPool{
			"test-pool": testPool,
		}

		Convey("移除空的池", func() {
			patches := gomonkey.ApplyMethod(reflect.TypeOf(testPool), "GetRunningCount", func(*task.TaskPool) int {
				return 0
			})
			patches.ApplyMethod(reflect.TypeOf(testPool), "GetWaitingCount", func(*task.TaskPool) int {
				return 0
			})
			defer patches.Reset()

			err := RemovePool("test-pool")
			So(err, ShouldBeNil)
			So(allPools, ShouldNotContainKey, "test-pool")
		})

		Convey("移除有任务的池", func() {
			patches := gomonkey.ApplyMethod(reflect.TypeOf(testPool), "GetRunningCount", func(*task.TaskPool) int {
				return 1
			})
			patches.ApplyMethod(reflect.TypeOf(testPool), "GetWaitingCount", func(*task.TaskPool) int {
				return 0
			})
			defer patches.Reset()

			err := RemovePool("test-pool")
			So(err, ShouldNotBeNil)
		})

		Convey("移除不存在的池", func() {
			err := RemovePool("non-existent-pool")
			So(err, ShouldBeNil)
		})
	})
}

func TestGetJob(t *testing.T) {
	Convey("测试GetJob函数", t, func() {
		testJob := &mockTaskJob{}
		allJobs = map[string]task.TaskJob{
			"job1": testJob,
		}

		Convey("获取存在的任务", func() {
			job, err := GetJob("job1")
			So(err, ShouldBeNil)
			So(job, ShouldEqual, testJob)
		})

		Convey("获取不存在的任务", func() {
			patches := gomonkey.ApplyFunc(task.LoadJob, func(uuri string) (task.TaskJob, error) {
				return nil, fmt.Errorf("not found")
			})
			defer patches.Reset()

			job, err := GetJob("non-existent-job")
			So(err, ShouldNotBeNil)
			So(job, ShouldBeNil)
		})

		Convey("从数据库加载失败", func() {
			patches := gomonkey.ApplyFunc(task.LoadJob, func(uuid string) (task.TaskJob, error) {
				return nil, fmt.Errorf("load failed")
			})
			defer patches.Reset()

			job, err := GetJob("job2")
			So(err, ShouldNotBeNil)
			So(job, ShouldBeNil)
		})
	})
}

func TestPoolNewJob(t *testing.T) {
	Convey("测试PoolNewJob函数", t, func() {
		testTaskRec := &dao.TaskRec{
			TaskObjRec: dao.TaskObjRec{UUID: "test-task"},
		}
		testJob := &mockTaskJob{}
		testPool := &task.TaskPool{}
		testPool.Pool.PoolId = "test-pool"

		allPools = map[string]*task.TaskPool{
			"test-pool": testPool,
		}

		Convey("成功创建任务", func() {
			patches := gomonkey.ApplyFunc(task.CreateJob, func(tr *dao.TaskRec) (task.TaskJob, error) {
				return testJob, nil
			})
			patches.ApplyFunc(selectPool, func(job task.TaskJob) (*task.TaskPool, error) {
				return testPool, nil
			})
			defer patches.Reset()

			job, err := PoolNewJob(testTaskRec)
			So(err, ShouldBeNil)
			So(job, ShouldEqual, testJob)
			So(allJobs, ShouldContainKey, "test-task")
		})

		Convey("创建任务失败", func() {
			patches := gomonkey.ApplyFunc(task.CreateJob, func(tr *dao.TaskRec) (task.TaskJob, error) {
				return nil, fmt.Errorf("create failed")
			})
			defer patches.Reset()

			job, err := PoolNewJob(testTaskRec)
			So(err, ShouldNotBeNil)
			So(job, ShouldBeNil)
		})

		Convey("选择池失败", func() {
			patches := gomonkey.ApplyFunc(task.CreateJob, func(tr *dao.TaskRec) (task.TaskJob, error) {
				return testJob, nil
			})
			patches.ApplyFunc(selectPool, func(job task.TaskJob) (*task.TaskPool, error) {
				return nil, fmt.Errorf("no suitable pool")
			})
			defer patches.Reset()

			job, err := PoolNewJob(testTaskRec)
			So(err, ShouldNotBeNil)
			So(job, ShouldBeNil)
		})
	})
}

func TestCancelJob(t *testing.T) {
	Convey("测试CancelJob函数", t, func() {
		testJob := &mockTaskJob{}
		allJobs = map[string]task.TaskJob{
			"job1": testJob,
		}

		Convey("取消存在的任务", func() {
			patches := gomonkey.ApplyFunc(stopJob, func(job task.TaskJob, status task.TaskStatus, err error) {
				// mock stopJob
			})
			defer patches.Reset()

			err := CancelJob("job1")
			So(err, ShouldBeNil)
		})

		Convey("取消不存在的任务", func() {
			patches := gomonkey.ApplyFunc(dao.ExistTask, func(uuid string) (bool, error) {
				return false, nil
			})
			defer patches.Reset()

			err := CancelJob("non-existent-job")
			So(err, ShouldNotBeNil)
		})

		Convey("检查任务存在性失败", func() {
			patches := gomonkey.ApplyFunc(dao.ExistTask, func(uuid string) (bool, error) {
				return false, fmt.Errorf("check failed")
			})
			defer patches.Reset()

			err := CancelJob("job2")
			So(err, ShouldNotBeNil)
		})
	})
}
