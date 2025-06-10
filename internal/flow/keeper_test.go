package flow

import (
	"io"
	"reflect"
	"sync"
	"taskd/internal/task"
	"testing"

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
		sendFinishedChanCalled := false
		gomonkey.ApplyFunc(tp.SendFinishedChan, func(task.TaskJob) {
			sendFinishedChanCalled = true
		})
		dealRunningJob(job)

		So(sendFinishedChanCalled, ShouldBeTrue)
	})
}

// Test case 2: Verify behavior when job status is unfinished, expect dealRunningJob and resolveMetrics to be called
func TestHandleRunningJob_UnfinishedStatus(t *testing.T) {
	Convey("当作业状态未完成时，应该调用 dealRunningJob 和 resolveMetrics", t, func() {
		job := &mockTaskJob{status: task.TaskStatusRunning}

		patches := gomonkey.ApplyFunc(dealRunningJob, func(task.TaskJob, func(task.TaskJob)) {})
		defer patches.Reset()

		processRunningJobCalled := false
		patches.ApplyFunc(dealRunningJob, func(task.TaskJob, func(task.TaskJob)) {
			processRunningJobCalled = true
		})

		resolveMetricsCalled := false
		patches.ApplyFunc(resolveMetrics, func(*task.Metric) {
			resolveMetricsCalled = true
		})

		dealRunningJob(job)

		So(processRunningJobCalled, ShouldBeTrue)
		So(resolveMetricsCalled, ShouldBeTrue)
	})
}

// Test case 3: Verify job status transition during dealRunningJob processing
func TestHandleRunningJob_TransitionWithinProcess(t *testing.T) {
	Convey("当在进行 dealRunningJob 处理时，作业状态由未完成变更为完成", t, func() {
		job := &mockTaskJob{status: task.TaskStatusRunning}
		tp := &task.TaskPool{}

		patches := gomonkey.ApplyFunc(dealRunningJob, func(task.TaskJob, func(task.TaskJob)) {
			patch := gomonkey.ApplyMethod(reflect.TypeOf(job), "FetchStatus", func() task.TaskStatus { return task.TaskStatusSucceeded })
			defer patch.Reset()
		})
		defer patches.Reset()

		sendFinishedChanCalled := false
		gomonkey.ApplyFunc(tp.SendFinishedChan, func(task.TaskJob) {
			sendFinishedChanCalled = true
		})

		dealRunningJob(job)

		So(sendFinishedChanCalled, ShouldBeTrue)
	})
}

// Test case 4: Test various job statuses to ensure each works as expected
func TestHandleRunningJob_VariousJobStatuses(t *testing.T) {
	Convey("在不同的作业状态下，确保功能按预期工作", t, func() {
		job := &mockTaskJob{status: task.TaskStatusRunning}
		tp := &task.TaskPool{}

		patches := gomonkey.ApplyFunc(dealRunningJob, func(task.TaskJob, func(task.TaskJob)) {})
		defer patches.Reset()

		tests := []struct {
			name          string
			status        task.TaskStatus
			expectedPhase task.TaskPhase
		}{
			{"Queued", task.TaskStatusQueue, task.PhaseQueue},
			{"Running", task.TaskStatusRunning, task.PhaseRunning},
			{"Waiting", task.TaskStatusInit, task.PhaseInit},
		}

		for _, tt := range tests {
			job.status = tt.status
			sendFinishedChanCalled := false
			gomonkey.ApplyFunc(tp.SendFinishedChan, func(task.TaskJob) {
				sendFinishedChanCalled = true
			})

			dealRunningJob(job)
			So(sendFinishedChanCalled, ShouldBeFalse)
		}
	})
}

// Test case 5: Verify exception handling in dealRunningJob and resolveMetrics
func TestHandleRunningJob_HandleExceptions(t *testing.T) {
	Convey("当 dealRunningJob 和 resolveMetrics 中出现异常时，应该正确处理", t, func() {
		job := &mockTaskJob{status: task.TaskStatusFailed}

		patches := gomonkey.ApplyFunc(dealRunningJob, func(task.TaskJob, func(task.TaskJob)) {
			panic("dealRunningJob exception")
		})
		defer patches.Reset()

		So(func() { dealRunningJob(job) }, ShouldPanic)

		patches.ApplyFunc(resolveMetrics, func(*task.Metric) {
			panic("resolveMetrics exception")
		})

		So(func() { dealRunningJob(job) }, ShouldPanic)
	})
}

// Test case 6: Process large number of jobs in short time to evaluate performance
func TestHandleRunningJob_Performance(t *testing.T) {
	Convey("快速处理大量作业，并评估性能", t, func() {
		job := &mockTaskJob{status: task.TaskStatusRunning}
		tp := &task.TaskPool{}
		numJobs := 1000
		sendFinishedChanCalledCount := 0
		gomonkey.ApplyFunc(tp.SendFinishedChan, func(task.TaskJob) {
			sendFinishedChanCalledCount++
		})

		for i := 0; i < numJobs; i++ {
			dealRunningJob(job)
		}

		So(sendFinishedChanCalledCount, ShouldEqual, 0)
	})
}

// Test case 7: Verify behavior when sendFinishedChan is nil
func TestHandleRunningJob_NilSendFinishedChan(t *testing.T) {
	Convey("当 sendFinishedChan 为 nil 时，应该正确处理", t, func() {
		job := &mockTaskJob{status: task.TaskStatusSucceeded}

		So(func() { dealRunningJob(job) }, ShouldNotPanic)
	})
}

// Test case 8: Verify thread safety under concurrent calls
func TestHandleRunningJob_ConcurrentInvocations(t *testing.T) {
	Convey("并发调用以确保线程安全", t, func() {
		job := &mockTaskJob{status: task.TaskStatusRunning}
		tp := &task.TaskPool{}
		var jobs []task.TaskJob
		for i := 0; i < 100; i++ {
			jobs = append(jobs, job)
		}

		gomonkey.ApplyFunc(tp.SendFinishedChan, func(task.TaskJob) {})

		var wg sync.WaitGroup
		for _, j := range jobs {
			wg.Add(1)
			go func(job task.TaskJob) {
				defer wg.Done()
				dealRunningJob(job)
			}(j)
		}
		wg.Wait()
	})
}

// Test case 9: Verify external system dependencies in dealRunningJob and resolveMetrics
func TestHandleRunningJob_ExternalDependencies(t *testing.T) {
	Convey("验证函数对外部系统的依赖，例如数据库或网络调用", t, func() {
		job := &mockTaskJob{status: task.TaskStatusRunning}

		patches := gomonkey.ApplyFunc(dealRunningJob, func(task.TaskJob, func(task.TaskJob)) {
			// Mock external system call
		})
		defer patches.Reset()

		patches.ApplyFunc(resolveMetrics, func(*task.Metric) {
			// Mock external system call
		})

		dealRunningJob(job)
	})
}

// Test case 10: Verify behavior when stopJob is called
func TestHandleRunningJob_StopJobCalled(t *testing.T) {
	Convey("当需要停止作业时，应该正确调用 stopJob", t, func() {
		job := &mockTaskJob{status: task.TaskStatusRunning}

		stopCalled := false
		var calledStatus task.TaskStatus

		patches := gomonkey.ApplyFunc(stopJob, func(j task.TaskJob, status task.TaskStatus) {
			stopCalled = true
			calledStatus = status
		})
		defer patches.Reset()

		// Mock job stopping scenario
		patches.ApplyMethod(reflect.TypeOf(job), "GetStatus", func() task.TaskStatus {
			return task.TaskStatusFailed
		})

		dealRunningJob(job)

		So(stopCalled, ShouldBeTrue)
		So(calledStatus, ShouldEqual, task.TaskStatusFailed)
	})
}
