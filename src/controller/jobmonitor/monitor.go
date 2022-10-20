//  Copyright Project Harbor Authors
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package jobmonitor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/goharbor/harbor/src/lib/orm"

	"github.com/goharbor/harbor/src/lib/log"

	"github.com/gocraft/work"

	"github.com/goharbor/harbor/src/common/job"
	"github.com/goharbor/harbor/src/lib/errors"
	"github.com/goharbor/harbor/src/lib/q"
	libRedis "github.com/goharbor/harbor/src/lib/redis"
	jm "github.com/goharbor/harbor/src/pkg/jobmonitor"
	"github.com/goharbor/harbor/src/pkg/task"
)

// All the jobs in the pool, or all pools
const All = "all"

// Ctl the controller instance of the worker pool controller
var Ctl = NewMonitorController()

// MonitorController defines the worker pool operations
type MonitorController interface {
	// ListPools lists the worker pools
	ListPools(ctx context.Context) ([]*jm.WorkerPool, error)
	// ListWorkers lists the workers in the pool
	ListWorkers(ctx context.Context, poolID string) ([]*jm.Worker, error)
	// StopRunningJob stop the running job
	StopRunningJob(ctx context.Context, jobID string) error
	// ListQueue lists job queues
	ListQueue(ctx context.Context) ([]*jm.Queue, error)
	// StopPendingJob stop the pending job
	StopPendingJob(ctx context.Context, jobType string) error
	// PauseJobQueues suspend the all schedules or resume the all schedules
	PauseJobQueues(ctx context.Context, jobType string, pause bool) error
	// SchedulerStatus get the job scheduler status
	SchedulerStatus(ctx context.Context) (bool, error)
}

type monitorController struct {
	poolManager           jm.PoolManager
	workerManager         jm.WorkerManager
	taskManager           task.Manager
	queueManager          jm.QueueManager
	monitorClient         func() (jm.JobServiceMonitorClient, error)
	jobServiceRedisClient func() (jm.RedisClient, error)
}

// NewWorkerPoolController ...
func NewMonitorController() MonitorController {
	return &monitorController{
		poolManager:           jm.NewPoolManager(),
		workerManager:         jm.NewWorkerManager(),
		taskManager:           task.NewManager(),
		queueManager:          jm.NewQueueClient(),
		monitorClient:         jobServiceMonitorClient,
		jobServiceRedisClient: jobServiceRedisClient,
	}
}

func (w *monitorController) StopRunningJob(ctx context.Context, jobID string) error {
	if strings.EqualFold(jobID, "all") {
		allRunningJobs, err := w.allRunningJobs(ctx)
		if err != nil {
			log.Errorf("failed to get all running jobs: %v", err)
			return err
		}
		for _, jobID := range allRunningJobs {
			if err := w.stopJob(ctx, jobID); err != nil {
				log.Errorf("failed to stop running job %s: %v", jobID, err)
				return err
			}
		}
		return nil
	}
	return w.stopJob(ctx, jobID)
}

func (w *monitorController) stopJob(ctx context.Context, jobID string) error {
	tasks, err := w.taskManager.List(ctx, &q.Query{Keywords: q.KeyWords{"job_id": jobID}})
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		return errors.BadRequestError(nil).WithMessage("job %s not found", jobID)
	}
	if len(tasks) != 1 {
		return fmt.Errorf("there are more than one task with the same job ID")
	}
	// use local transaction to avoid rollback batch success tasks to previous state when one fail
	if ctx == nil {
		log.Debug("context is nil, skip stop operation")
		return nil
	}
	return orm.WithTransaction(func(ctx context.Context) error {
		return w.taskManager.Stop(ctx, tasks[0].ID)
	})(orm.SetTransactionOpNameToContext(ctx, "tx-stop-job"))
}

func (w *monitorController) allRunningJobs(ctx context.Context) ([]string, error) {
	jobIDs := make([]string, 0)
	wks, err := w.ListWorkers(ctx, All)
	if err != nil {
		log.Errorf("failed to list workers: %v", err)
		return nil, err
	}
	for _, wk := range wks {
		jobIDs = append(jobIDs, wk.JobID)
	}
	return jobIDs, nil
}

func (w *monitorController) ListQueue(ctx context.Context) ([]*jm.Queue, error) {
	mClient, err := w.monitorClient()
	if err != nil {
		return nil, err
	}
	qs, err := mClient.Queues()
	if err != nil {
		return nil, err
	}
	redisClient, err := w.jobServiceRedisClient()
	if err != nil {
		return nil, err
	}
	// the original queue doesn't include the paused status, fetch it from the redis
	statusMap, err := redisClient.AllJobTypeStatus(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*jm.Queue, 0)
	for _, q := range qs {
		result = append(result, &jm.Queue{
			JobType: q.JobName,
			Count:   q.Count,
			Latency: q.Latency,
			Paused:  statusMap[q.JobName],
		})
	}
	return result, nil
}

func (w *monitorController) StopPendingJob(ctx context.Context, jobType string) error {
	redisClient, err := w.jobServiceRedisClient()
	if err != nil {
		return err
	}
	if strings.EqualFold(jobType, "all") {
		jobTypes, err := redisClient.AllJobTypes(ctx)
		if err != nil {
			return err
		}
		for _, jobType := range jobTypes {
			if err := w.StopPendingJob(ctx, jobType); err != nil {
				return err
			}
		}
		return nil
	}
	jobIDs, err := redisClient.StopPendingJobs(ctx, jobType)
	if err != nil {
		return err
	}
	return w.UpdateJobStatusInTask(ctx, jobIDs)
}

func (w *monitorController) UpdateJobStatusInTask(ctx context.Context, jobIDs []string) error {
	for _, jobID := range jobIDs {
		ts, err := w.taskManager.List(ctx, q.New(q.KeyWords{"job_id": jobID}))
		if err != nil {
			return err
		}
		if len(ts) == 0 {
			continue
		}
		ts[0].Status = "Stopped"
		if err := w.taskManager.Update(ctx, ts[0], "Status"); err != nil {
			return err
		}
	}
	return nil
}

func jobServiceMonitorClient() (jm.JobServiceMonitorClient, error) {
	cfg, err := job.GlobalClient.GetJobServiceConfig()
	if err != nil {
		return nil, err
	}
	config := cfg.RedisPoolConfig
	pool, err := libRedis.GetRedisPool("JobService", config.RedisURL, &libRedis.PoolParam{
		PoolMaxIdle:     0,
		PoolIdleTimeout: time.Duration(config.IdleTimeoutSecond) * time.Second,
	})
	if err != nil {
		log.Errorf("failed to get redis pool: %v", err)
		return nil, err
	}
	return work.NewClient(fmt.Sprintf("{%s}", config.Namespace), pool), nil
}

func jobServiceRedisClient() (jm.RedisClient, error) {
	cfg, err := job.GlobalClient.GetJobServiceConfig()
	if err != nil {
		return nil, err
	}
	config := cfg.RedisPoolConfig
	return jm.NewRedisClient(config)
}

func (w *monitorController) ListWorkers(ctx context.Context, poolID string) ([]*jm.Worker, error) {
	mClient, err := w.monitorClient()
	if err != nil {
		return nil, err
	}
	return w.workerManager.List(ctx, mClient, poolID)
}

func (w *monitorController) ListPools(ctx context.Context) ([]*jm.WorkerPool, error) {
	mClient, err := w.monitorClient()
	if err != nil {
		return nil, err
	}
	return w.poolManager.List(ctx, mClient)
}

func (w *monitorController) PauseJobQueues(ctx context.Context, jobType string, pause bool) error {
	redisClient, err := w.jobServiceRedisClient()
	if err != nil {
		return err
	}
	if strings.EqualFold(jobType, "all") {
		jobTypes, err := redisClient.AllJobTypes(ctx)
		if err != nil {
			return err
		}
		for _, jobType := range jobTypes {
			if err := w.PauseJobQueues(ctx, jobType, pause); err != nil {
				return err
			}
		}
		return nil
	}
	if pause {
		return redisClient.PauseJob(ctx, jobType)
	}
	return redisClient.UnpauseJob(ctx, jobType)
}

func (w *monitorController) SchedulerStatus(ctx context.Context) (bool, error) {
	redisClient, err := w.jobServiceRedisClient()
	if err != nil {
		return false, err
	}
	statusMap, err := redisClient.AllJobTypeStatus(ctx)
	if err != nil {
		return false, err
	}
	return statusMap["SCHEDULER"], nil
}
