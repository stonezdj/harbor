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
	"github.com/goharbor/harbor/src/common/job"
	"github.com/goharbor/harbor/src/lib/q"
	jm "github.com/goharbor/harbor/src/pkg/jobmonitor"
	"github.com/goharbor/harbor/src/pkg/scheduler"
	"github.com/goharbor/harbor/src/pkg/task"
	"strings"
)

// Ctl the controller instance of the worker pool controller
var Ctl = NewWorkerPoolController()

// WorkerPoolController defines the worker pool operations
type WorkerPoolController interface {
	// List lists the worker pools
	List(ctx context.Context) ([]*jm.WorkerPool, error)
	// ListWorker lists the workers in the pool
	ListWorker(ctx context.Context, poolID string) ([]*jm.Worker, error)
	// ListQueue lists job queues
	ListQueue(ctx context.Context) ([]*jm.Queue, error)
	// StopRunningJob stop the running job
	StopRunningJob(ctx context.Context, jobID string) error
	// StopPendingJob stop the pending job
	StopPendingJob(ctx context.Context, jobType string) error
	// ListSchedule list the schedules in the job service
	ListSchedule(ctx context.Context) ([]*scheduler.Schedule, error)
	// PauseJobQueues suspend the all schedules or resume the all schedules
	PauseJobQueues(ctx context.Context, jobType string, pause bool) error
	// GetSchedulerStatus get the job scheduler status
	GetSchedulerStatus(ctx context.Context) (bool, error)
}

type workerPoolController struct {
	poolManager      jm.PoolManager
	jobServiceClient jm.JobServiceClient
	taskManager      task.Manager
	sch              scheduler.Scheduler
}

func (w *workerPoolController) GetSchedulerStatus(ctx context.Context) (bool, error) {
	cfg, err := job.GlobalClient.GetJobConfig()
	if err != nil {
		return false, err
	}
	statusMap, err := w.jobServiceClient.AllJobTypeStatus(ctx, cfg.RedisPoolConfig)
	if err != nil {
		return false, err
	}
	return statusMap["SCHEDULER"], nil
}

func (w *workerPoolController) PauseJobQueues(ctx context.Context, jobType string, pause bool) error {
	cfg, err := job.GlobalClient.GetJobConfig()
	if err != nil {
		return err
	}
	if strings.EqualFold(jobType, "all") {
		jobTypes, err := w.jobServiceClient.AllJobTypes(ctx, cfg.RedisPoolConfig)
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
		return w.jobServiceClient.PauseJob(ctx, cfg.RedisPoolConfig, jobType)
	}
	return w.jobServiceClient.UnpauseJob(ctx, cfg.RedisPoolConfig, jobType)
}

func (w *workerPoolController) ListSchedule(ctx context.Context) ([]*scheduler.Schedule, error) {
	return w.sch.ListSchedules(ctx, nil)
}

func (w *workerPoolController) StopPendingJob(ctx context.Context, jobType string) error {
	cfg, err := job.GlobalClient.GetJobConfig()
	if err != nil {
		return err
	}
	if strings.EqualFold(jobType, "all") {
		jobTypes, err := w.jobServiceClient.AllJobTypes(ctx, cfg.RedisPoolConfig)
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
	return w.jobServiceClient.StopPendingJobs(ctx, cfg.RedisPoolConfig, jobType)
}

func (w *workerPoolController) ListQueue(ctx context.Context) ([]*jm.Queue, error) {
	cfg, err := job.GlobalClient.GetJobConfig()
	if err != nil {
		return nil, err
	}
	qs, err := w.jobServiceClient.ListQueues(ctx, cfg.RedisPoolConfig)
	if err != nil {
		return nil, err
	}
	statusMap, err := w.jobServiceClient.AllJobTypeStatus(ctx, cfg.RedisPoolConfig)
	if err != nil {
		return nil, err
	}
	for _, q := range qs {
		q.Paused = statusMap[q.JobType]
	}
	return qs, nil
}

func (w *workerPoolController) StopRunningJob(ctx context.Context, jobID string) error {
	if strings.EqualFold(jobID, "all") {
		allRunningJobs, err := w.allRunningJobs(ctx)
		if err != nil {
			return err
		}
		for _, jobID := range allRunningJobs {
			if err := w.StopRunningJob(ctx, jobID); err != nil {
				return err
			}
		}
		return nil
	}
	tasks, err := w.taskManager.List(ctx, &q.Query{Keywords: q.KeyWords{"job_id": jobID}})
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		return nil
	}
	if len(tasks) != 1 {
		return fmt.Errorf("there are more than one task with the same job ID")
	}
	return w.taskManager.Stop(ctx, tasks[0].ID)
}

func (w *workerPoolController) allRunningJobs(ctx context.Context) ([]string, error) {
	jobIDs := make([]string, 0)
	wks, err := w.ListWorker(ctx, "all")
	if err != nil {
		return nil, err
	}
	for _, wk := range wks {
		jobIDs = append(jobIDs, wk.JobID)
	}
	return jobIDs, nil
}

func (w *workerPoolController) ListWorker(ctx context.Context, poolID string) ([]*jm.Worker, error) {
	cfg, err := job.GlobalClient.GetJobConfig()
	if err != nil {
		return nil, err
	}
	return w.jobServiceClient.ListWorkers(ctx, cfg.RedisPoolConfig, poolID)
}

func (w *workerPoolController) List(ctx context.Context) ([]*jm.WorkerPool, error) {
	cfg, err := job.GlobalClient.GetJobConfig()
	if err != nil {
		return nil, err
	}
	return w.poolManager.List(ctx, cfg.RedisPoolConfig)
}

// NewWorkerPoolController ...
func NewWorkerPoolController() WorkerPoolController {
	return &workerPoolController{poolManager: jm.NewPoolManager(), jobServiceClient: jm.NewJobServiceClient(), taskManager: task.NewManager(), sch: scheduler.New()}
}
