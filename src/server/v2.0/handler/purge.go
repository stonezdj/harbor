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
	"github.com/goharbor/harbor/src/common"
	"github.com/goharbor/harbor/src/common/rbac"
	"github.com/goharbor/harbor/src/controller/jobservice"
	pg "github.com/goharbor/harbor/src/controller/purge"
	"github.com/goharbor/harbor/src/lib/errors"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/lib/q"
	"github.com/goharbor/harbor/src/pkg/task"
	"github.com/goharbor/harbor/src/server/v2.0/handler/model"
	"github.com/goharbor/harbor/src/server/v2.0/models"
	"github.com/goharbor/harbor/src/server/v2.0/restapi/operations/purge"
	operation "github.com/goharbor/harbor/src/server/v2.0/restapi/operations/purge"
	"strings"
)

type purgeAPI struct {
	BaseAPI
	purgeCtr     pg.Controller
	schedulerCtl jobservice.SchedulerController
	taskCtl      jobservice.TaskController
	executionCtl jobservice.ExecutionController
}

func newPurgeAPI() *purgeAPI {
	return &purgeAPI{
		purgeCtr:     pg.Ctrl,
		schedulerCtl: jobservice.SchedulerCtl,
		taskCtl:      jobservice.TaskCtl,
		executionCtl: jobservice.ExecutionCtl,
	}
}

func (p *purgeAPI) CreatePurgeSchedule(ctx context.Context, params purge.CreatePurgeScheduleParams) middleware.Responder {
	if err := p.RequireSystemAccess(ctx, rbac.ActionCreate, rbac.ResourcePurgeAuditLog); err != nil {
		return p.SendError(ctx, err)
	}
	id, err := p.kick(ctx, params.Schedule.Schedule.Type, params.Schedule.Schedule.Cron, params.Schedule.Parameters)
	if err != nil {
		return p.SendError(ctx, err)
	}
	lastSlashIndex := strings.LastIndex(params.HTTPRequest.URL.Path, "/")
	if lastSlashIndex != -1 {
		location := fmt.Sprintf("%s/%d", params.HTTPRequest.URL.Path[:lastSlashIndex], id)
		return purge.NewCreatePurgeScheduleCreated().WithLocation(location)
	}
	return purge.NewCreatePurgeScheduleCreated()
}

func (p *purgeAPI) kick(ctx context.Context, scheType string, cron string, parameters map[string]interface{}) (int64, error) {
	if parameters == nil {
		parameters = make(map[string]interface{})
	}
	var err error
	var id int64

	policy := pg.JobPolicy{
		ExtraAttrs: parameters,
	}
	if dryRun, ok := parameters[common.PurgeAuditDryRun].(bool); ok {
		policy.DryRun = dryRun
	}
	if includeOperations, ok := parameters[common.PurgeAuditIncludeOperations].(string); ok {
		policy.IncludeOperations = includeOperations
	}
	if retentionHour, ok := parameters[common.PurgeAuditRetentionHour]; ok {
		if rh, ok := retentionHour.(json.Number); ok {
			ret, err := rh.Int64()
			if err != nil {
				log.Warning("failed to convert audit_retention_hour, %v", err)
			}
			policy.RetentionHour = int(ret)
		}
	}

	switch scheType {
	case ScheduleManual:
		id, err = p.purgeCtr.Start(ctx, policy, task.ExecutionTriggerManual)
	case ScheduleNone:
		//delete the schedule of purge
		err = p.schedulerCtl.DeleteSchedule(ctx, pg.VendorType)
	case ScheduleHourly, ScheduleDaily, ScheduleWeekly, ScheduleCustom:
		err = p.updateSchedule(ctx, scheType, cron, policy, parameters)
	}
	return id, err
}

func (p *purgeAPI) updateSchedule(ctx context.Context, cronType, cron string, policy pg.JobPolicy, extraParams map[string]interface{}) error {
	if err := p.schedulerCtl.DeleteSchedule(ctx, pg.VendorType); err != nil {
		return err
	}
	return p.createSchedule(ctx, cronType, cron, policy, extraParams)
}

func (p *purgeAPI) GetPurgeHistory(ctx context.Context, params purge.GetPurgeHistoryParams) middleware.Responder {
	if err := p.RequireSystemAccess(ctx, rbac.ActionRead, rbac.ResourcePurgeAuditLog); err != nil {
		return p.SendError(ctx, err)
	}
	query, err := p.BuildQuery(ctx, params.Q, params.Sort, params.Page, params.PageSize)
	if err != nil {
		return p.SendError(ctx, err)
	}
	total, err := p.executionCtl.ExecutionCount(ctx, pg.VendorType, query)
	if err != nil {
		return p.SendError(ctx, err)
	}
	execs, err := p.executionCtl.ListExecutions(ctx, pg.VendorType, query)
	if err != nil {
		p.SendError(ctx, err)
	}

	var hs []*model.ExecHistory
	for _, exec := range execs {
		extraAttrsString, err := json.Marshal(exec.ExtraAttrs)
		if err != nil {
			return p.SendError(ctx, err)
		}
		hs = append(hs, &model.ExecHistory{
			ID:         exec.ID,
			Name:       pg.VendorType,
			Kind:       exec.Trigger,
			Parameters: string(extraAttrsString),
			Schedule: &model.ScheduleParam{
				Type: exec.Trigger,
			},
			Status:       exec.Status,
			CreationTime: exec.StartTime,
			UpdateTime:   exec.EndTime,
		})
	}
	var results []*models.ExecHistory
	for _, h := range hs {
		results = append(results, h.ToSwagger())
	}

	return operation.NewGetPurgeHistoryOK().
		WithXTotalCount(total).
		WithLink(p.Links(ctx, params.HTTPRequest.URL, total, query.PageNumber, query.PageSize).String()).
		WithPayload(results)
}

func (p *purgeAPI) GetPurgeJob(ctx context.Context, params purge.GetPurgeJobParams) middleware.Responder {
	if err := p.RequireSystemAccess(ctx, rbac.ActionRead, rbac.ResourceGarbageCollection); err != nil {
		return p.SendError(ctx, err)
	}
	exec, err := p.executionCtl.GetExecution(ctx, pg.VendorType, params.PurgeID)
	if err != nil {
		return p.SendError(ctx, err)
	}

	extraAttrsString, err := json.Marshal(exec.ExtraAttrs)
	if err != nil {
		return p.SendError(ctx, err)
	}

	res := &model.ExecHistory{
		ID:         exec.ID,
		Name:       pg.VendorType,
		Kind:       exec.Trigger,
		Parameters: string(extraAttrsString),
		Status:     exec.Status,
		Schedule: &model.ScheduleParam{
			Type: exec.Trigger,
		},
		CreationTime: exec.StartTime,
		UpdateTime:   exec.EndTime,
	}
	res.ToSwagger()

	return operation.NewGetPurgeJobOK().WithPayload(res.ToSwagger())
}

func (p *purgeAPI) GetPurgeJobLog(ctx context.Context, params purge.GetPurgeJobLogParams) middleware.Responder {
	if err := p.RequireSystemAccess(ctx, rbac.ActionRead, rbac.ResourceGarbageCollection); err != nil {
		return p.SendError(ctx, err)
	}
	tasks, err := p.taskCtl.ListTasks(ctx, pg.VendorType, q.New(q.KeyWords{
		"ExecutionID": params.PurgeID,
	}))
	if err != nil {
		return p.SendError(ctx, err)
	}
	if len(tasks) == 0 {
		return p.SendError(ctx,
			errors.New(nil).WithCode(errors.NotFoundCode).
				WithMessage("garbage collection %d taskLog is not found", params.PurgeID))
	}
	taskLog, err := p.taskCtl.GetTaskLog(ctx, pg.VendorType, tasks[0].ID)
	if err != nil {
		return p.SendError(ctx, err)
	}
	return operation.NewGetPurgeJobLogOK().WithPayload(string(taskLog))
}

func (p *purgeAPI) GetPurgeSchedule(ctx context.Context, params purge.GetPurgeScheduleParams) middleware.Responder {
	if err := p.RequireSystemAccess(ctx, rbac.ActionRead, rbac.ResourcePurgeAuditLog); err != nil {
		return p.SendError(ctx, err)
	}
	schedule, err := p.schedulerCtl.GetSchedule(ctx, pg.VendorType)
	if errors.IsNotFoundErr(err) {
		return operation.NewGetPurgeScheduleOK()
	}
	if err != nil {
		return p.SendError(ctx, err)
	}
	execHistory := &models.ExecHistory{
		ID:            schedule.ID,
		JobName:       "",
		JobKind:       schedule.CRON,
		JobParameters: pg.String(schedule.ExtraAttrs),
		Deleted:       false,
		JobStatus:     schedule.Status,
		Schedule: &models.ScheduleObj{
			Cron: schedule.CRON,
			Type: schedule.CRONType,
		},
		CreationTime: strfmt.DateTime(schedule.CreationTime),
		UpdateTime:   strfmt.DateTime(schedule.UpdateTime),
	}
	return operation.NewGetPurgeScheduleOK().WithPayload(execHistory)
}

func (p *purgeAPI) UpdatePurgeSchedule(ctx context.Context, params purge.UpdatePurgeScheduleParams) middleware.Responder {
	if err := p.RequireSystemAccess(ctx, rbac.ActionUpdate, rbac.ResourceGarbageCollection); err != nil {
		return p.SendError(ctx, err)
	}
	_, err := p.kick(ctx, params.Schedule.Schedule.Type, params.Schedule.Schedule.Cron, params.Schedule.Parameters)
	if err != nil {
		return p.SendError(ctx, err)
	}
	return operation.NewUpdatePurgeScheduleOK()
}

func (p *purgeAPI) createSchedule(ctx context.Context, cronType string, cron string, policy pg.JobPolicy, extraParam map[string]interface{}) error {
	if cron == "" {
		return errors.New(nil).WithCode(errors.BadRequestCode).
			WithMessage("empty cron string for schedule")
	}
	_, err := p.schedulerCtl.CreateSchedule(ctx, pg.VendorType, cronType, cron, pg.SchedulerCallback, policy, extraParam)
	if err != nil {
		return err
	}
	return nil
}
