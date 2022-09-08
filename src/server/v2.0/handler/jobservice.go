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

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/goharbor/harbor/src/common/rbac"
	"github.com/goharbor/harbor/src/controller/jobmonitor"
	"github.com/goharbor/harbor/src/lib/errors"
	"github.com/goharbor/harbor/src/lib/log"
	jm "github.com/goharbor/harbor/src/pkg/jobmonitor"
	"github.com/goharbor/harbor/src/pkg/scheduler"
	"github.com/goharbor/harbor/src/server/v2.0/models"
	"github.com/goharbor/harbor/src/server/v2.0/restapi/operations/jobservice"
	"strings"
	"time"
)

type jobServiceAPI struct {
	BaseAPI
	jobCtr jobmonitor.WorkerPoolController
}

func (j *jobServiceAPI) GetSchedulerStatus(ctx context.Context, params jobservice.GetSchedulerStatusParams) middleware.Responder {
	if err := j.RequireAuthenticated(ctx); err != nil {
		return j.SendError(ctx, err)
	}
	paused, err := j.jobCtr.GetSchedulerStatus(ctx)
	if err != nil {
		return j.SendError(ctx, err)
	}
	return jobservice.NewGetSchedulerStatusOK().WithPayload(&models.SchedulerStatus{
		Paused: paused,
	})
}

func newJobServiceAPI() *jobServiceAPI {
	return &jobServiceAPI{jobCtr: jobmonitor.Ctl}
}

func (j *jobServiceAPI) ListSchedules(ctx context.Context, params jobservice.ListSchedulesParams) middleware.Responder {
	if err := j.RequireSystemAccess(ctx, rbac.ActionList, rbac.ResourceJobServiceDashBoard); err != nil {
		return j.SendError(ctx, err)
	}
	schs, err := j.jobCtr.ListSchedule(ctx)
	if err != nil {
		return j.SendError(ctx, err)
	}
	return jobservice.NewListSchedulesOK().WithPayload(toScheduleResponse(schs))
}

func toScheduleResponse(schs []*scheduler.Schedule) []*models.ScheduleTask {
	result := make([]*models.ScheduleTask, 0)
	for _, s := range schs {
		extraAttr := []byte("")
		if s.ExtraAttrs != nil {
			extra, err := json.Marshal(s.ExtraAttrs)
			if err != nil {
				log.Warningf("failed to extract extra attribute, error %v", err)
			} else {
				extraAttr = extra
			}
		}
		result = append(result, &models.ScheduleTask{
			ID:           s.ID,
			VendorType:   s.VendorType,
			VendorID:     s.VendorID,
			ExtraAttrs:   string(extraAttr),
			CreationTime: strfmt.DateTime(s.CreationTime),
		})
	}
	return result
}

func (j *jobServiceAPI) ListJobQueues(ctx context.Context, params jobservice.ListJobQueuesParams) middleware.Responder {
	if err := j.RequireSystemAccess(ctx, rbac.ActionList, rbac.ResourceJobServiceDashBoard); err != nil {
		return j.SendError(ctx, err)
	}
	queues, err := j.jobCtr.ListQueue(ctx)
	if err != nil {
		return j.SendError(ctx, err)
	}
	return jobservice.NewListJobQueuesOK().WithPayload(toQueueResponse(queues))
}

func toQueueResponse(queues []*jm.Queue) []*models.JobQueue {
	result := make([]*models.JobQueue, 0)
	for _, q := range queues {
		result = append(result, &models.JobQueue{
			JobType: q.JobType,
			Count:   q.Count,
			Latency: q.Latency,
			Paused:  q.Paused,
		})
	}
	return result
}

func (j *jobServiceAPI) ActionPendingJobs(ctx context.Context, params jobservice.ActionPendingJobsParams) middleware.Responder {
	if err := j.RequireSystemAccess(ctx, rbac.ActionStop, rbac.ResourceJobServiceDashBoard); err != nil {
		return j.SendError(ctx, err)
	}
	jobType := strings.ToUpper(params.JobType)
	action := strings.ToLower(params.ActionRequest.Action)
	if !strings.EqualFold(action, "stop") && !strings.EqualFold(action, "resume") && !strings.EqualFold(action, "pause") {
		return j.SendError(ctx, errors.BadRequestError(fmt.Errorf("the action is not supported")))
	}
	if strings.EqualFold(action, "stop") {
		err := j.jobCtr.StopPendingJob(ctx, jobType)
		if err != nil {
			return j.SendError(ctx, err)
		}
	}
	if strings.EqualFold(action, "pause") || strings.EqualFold(action, "resume") {
		err := j.jobCtr.PauseJobQueues(ctx, jobType, strings.EqualFold(action, "pause"))
		if err != nil {
			return j.SendError(ctx, err)
		}
	}
	return jobservice.NewActionPendingJobsOK()
}

func (j *jobServiceAPI) GetWorkerPools(ctx context.Context, params jobservice.GetWorkerPoolsParams) middleware.Responder {
	if err := j.RequireSystemAccess(ctx, rbac.ActionList, rbac.ResourceJobServiceDashBoard); err != nil {
		return j.SendError(ctx, err)
	}
	workPools, err := j.jobCtr.List(ctx)
	if err != nil {
		return j.SendError(ctx, err)
	}
	return jobservice.NewGetWorkerPoolsOK().WithPayload(toWorkerPoolResponse(workPools))
}

func (j *jobServiceAPI) GetWorkers(ctx context.Context, params jobservice.GetWorkersParams) middleware.Responder {
	if err := j.RequireSystemAccess(ctx, rbac.ActionList, rbac.ResourceJobServiceDashBoard); err != nil {
		return j.SendError(ctx, err)
	}
	workPools, err := j.jobCtr.ListWorker(ctx, params.PoolID)
	if err != nil {
		return j.SendError(ctx, err)
	}
	return jobservice.NewGetWorkersOK().WithPayload(toWorkerResponse(workPools))
}

func (j *jobServiceAPI) StopRunningJob(ctx context.Context, params jobservice.StopRunningJobParams) middleware.Responder {
	if err := j.RequireSystemAccess(ctx, rbac.ActionStop, rbac.ResourceJobServiceDashBoard); err != nil {
		return j.SendError(ctx, err)
	}
	err := j.jobCtr.StopRunningJob(ctx, params.JobID)
	if err != nil {
		return j.SendError(ctx, err)
	}
	return jobservice.NewStopRunningJobOK()
}

func toWorkerResponse(wks []*jm.Worker) []*models.Worker {
	workers := make([]*models.Worker, 0)
	for _, w := range wks {
		p := &models.Worker{
			ID:        w.WorkerID,
			JobName:   w.JobName,
			JobID:     w.JobID,
			PoolID:    w.PoolID,
			Args:      w.Args,
			StartAt:   covertTime(w.StartedAt),
			CheckinAt: covertTime(w.CheckInAt),
		}
		workers = append(workers, p)
	}
	return workers
}

func toWorkerPoolResponse(wps []*jm.WorkerPool) []*models.WorkerPool {
	pools := make([]*models.WorkerPool, 0)
	for _, wp := range wps {
		p := &models.WorkerPool{
			Pid:          int64(wp.PID),
			HeartbeatAt:  covertTime(wp.HeartbeatAt),
			Concurrency:  int64(wp.Concurency),
			WorkerPoolID: wp.PoolID,
			StartAt:      covertTime(wp.StartAt),
		}
		pools = append(pools, p)
	}
	return pools
}

func covertTime(t int64) strfmt.DateTime {
	uxt := time.Unix(int64(t), 0)
	return strfmt.DateTime(uxt)
}
