// Copyright Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dao

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/goharbor/harbor/src/jobservice/job"
	"github.com/goharbor/harbor/src/lib/errors"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/lib/orm"
	"github.com/goharbor/harbor/src/lib/q"
)

// ExecutionDAO is the data access object interface for execution
type ExecutionDAO interface {
	// Count returns the total count of executions according to the query
	// Query the "ExtraAttrs" by setting 'query.Keywords["ExtraAttrs.key"]="value"'
	Count(ctx context.Context, query *q.Query) (count int64, err error)
	// List the executions according to the query
	// Query the "ExtraAttrs" by setting 'query.Keywords["ExtraAttrs.key"]="value"'
	List(ctx context.Context, query *q.Query) (executions []*Execution, err error)
	// Get the specified execution
	Get(ctx context.Context, id int64) (execution *Execution, err error)
	// Create an execution
	Create(ctx context.Context, execution *Execution) (id int64, err error)
	// Update the specified execution. Only the properties specified by "props" will be updated if it is set
	Update(ctx context.Context, execution *Execution, props ...string) (err error)
	// Delete the specified execution
	Delete(ctx context.Context, id int64) (err error)
	// GetMetrics returns the task metrics for the specified execution
	GetMetrics(ctx context.Context, id int64) (metrics *Metrics, err error)
	// GetEndTime get the execution complete time, get it from the max complete task update time.
	GetEndTime(ctx context.Context, executionID int64) (*time.Time, error)
}

// NewExecutionDAO returns an instance of ExecutionDAO
func NewExecutionDAO() ExecutionDAO {
	return &executionDAO{
		taskDAO: NewTaskDAO(),
	}
}

type executionDAO struct {
	taskDAO TaskDAO
}

func (e *executionDAO) Count(ctx context.Context, query *q.Query) (int64, error) {
	if query != nil {
		// ignore the page number and size
		query = &q.Query{
			Keywords: query.Keywords,
		}
	}
	qs, err := e.querySetter(ctx, query)
	if err != nil {
		return 0, err
	}
	return qs.Count()
}

func (e *executionDAO) List(ctx context.Context, query *q.Query) ([]*Execution, error) {
	executions := []*Execution{}
	qs, err := e.querySetter(ctx, query)
	if err != nil {
		return nil, err
	}
	if _, err = qs.All(&executions); err != nil {
		return nil, err
	}
	return executions, nil
}

func (e *executionDAO) Get(ctx context.Context, id int64) (*Execution, error) {
	execution := &Execution{
		ID: id,
	}
	ormer, err := orm.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := ormer.Read(execution); err != nil {
		if e := orm.AsNotFoundError(err, "execution %d not found", id); e != nil {
			err = e
		}
		return nil, err
	}
	return execution, nil
}

func (e *executionDAO) Create(ctx context.Context, execution *Execution) (int64, error) {
	ormer, err := orm.FromContext(ctx)
	if err != nil {
		return 0, err
	}
	return ormer.Insert(execution)
}

func (e *executionDAO) Update(ctx context.Context, execution *Execution, props ...string) error {
	ormer, err := orm.FromContext(ctx)
	if err != nil {
		return err
	}
	n, err := ormer.Update(execution, props...)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.NotFoundError(nil).WithMessage("execution %d not found", execution.ID)
	}
	return nil
}

func (e *executionDAO) Delete(ctx context.Context, id int64) error {
	ormer, err := orm.FromContext(ctx)
	if err != nil {
		return err
	}
	n, err := ormer.Delete(&Execution{
		ID: id,
	})
	if err != nil {
		if e := orm.AsForeignKeyError(err,
			"the execution %d is referenced by other resources", id); e != nil {
			err = e
		}
		return err
	}
	if n == 0 {
		return errors.NotFoundError(nil).WithMessage("execution %d not found", id)
	}
	return nil
}

func (e *executionDAO) GetMetrics(ctx context.Context, id int64) (*Metrics, error) {
	scs, err := e.taskDAO.ListStatusCount(ctx, id)
	if err != nil {
		return nil, err
	}
	metrics := &Metrics{}
	if len(scs) == 0 {
		return metrics, nil
	}

	for _, sc := range scs {
		switch sc.Status {
		case job.SuccessStatus.String():
			metrics.SuccessTaskCount = sc.Count
		case job.ErrorStatus.String():
			metrics.ErrorTaskCount = sc.Count
		case job.PendingStatus.String():
			metrics.PendingTaskCount = sc.Count
		case job.RunningStatus.String():
			metrics.RunningTaskCount = sc.Count
		case job.ScheduledStatus.String():
			metrics.ScheduledTaskCount = sc.Count
		case job.StoppedStatus.String():
			metrics.StoppedTaskCount = sc.Count
		default:
			log.Errorf("unknown task status: %s", sc.Status)
		}
	}
	metrics.TaskCount = metrics.SuccessTaskCount + metrics.ErrorTaskCount +
		metrics.PendingTaskCount + metrics.RunningTaskCount +
		metrics.ScheduledTaskCount + metrics.StoppedTaskCount
	return metrics, nil
}

// GetEndTime get the execution complete time
func (e *executionDAO) GetEndTime(ctx context.Context, executionID int64) (*time.Time, error) {
	endTime := &time.Time{}
	ormer, err := orm.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := ormer.Raw("select max(end_time) from task where execution_id = ? and status in ('Success', 'Stopped', 'Error')", executionID).QueryRow(&endTime); err != nil {
		return nil, err
	}
	return endTime, nil
}

func (e *executionDAO) querySetter(ctx context.Context, query *q.Query) (orm.QuerySeter, error) {
	qs, err := orm.QuerySetter(ctx, &Execution{}, query)
	if err != nil {
		return nil, err
	}

	// append the filter for "extra attrs"
	if query != nil && len(query.Keywords) > 0 {
		var (
			key       string
			keyPrefix string
			value     interface{}
		)
		for key, value = range query.Keywords {
			if strings.HasPrefix(key, "ExtraAttrs.") {
				keyPrefix = "ExtraAttrs."
				break
			}
			if strings.HasPrefix(key, "extra_attrs.") {
				keyPrefix = "extra_attrs."
				break
			}
		}
		if len(keyPrefix) == 0 || keyPrefix == key {
			return qs, nil
		}

		// key with keyPrefix supports multi-level query operator on PostgreSQL JSON data
		// examples:
		// key = extra_attrs.id,
		//  ==> sql = "select id from execution where extra_attrs->>?=?", args = {id, value}
		// key = extra_attrs.artifact.digest
		//  ==> sql = "select id from execution where extra_attrs->?->>?=?", args = {artifact, id, value}
		// key = extra_attrs.a.b.c
		//  ==> sql = "select id from execution where extra_attrs->?->?->>?=?", args = {a, b, c, value}
		keys := strings.Split(strings.TrimPrefix(key, keyPrefix), ".")
		var args []interface{}
		for _, item := range keys {
			args = append(args, item)
		}
		args = append(args, value)
		inClause, err := orm.CreateInClause(ctx, buildInClauseSQLForExtraAttrs(keys), args...)
		if err != nil {
			return nil, err
		}
		qs = qs.FilterRaw("id", inClause)
	}

	return qs, nil
}

// Param keys is strings.Split() after trim "extra_attrs."/"ExtraAttrs." prefix
func buildInClauseSQLForExtraAttrs(keys []string) string {
	switch len(keys) {
	case 0:
		// won't fall into this case, as the if condition on "keyPrefix == key"
		// act as a place holder to ensure "default" is equivalent to "len(keys) >= 2"
		return ""
	case 1:
		return "select id from execution where extra_attrs->>?=?"
	default:
		// len(keys) >= 2
		elements := make([]string, len(keys)-1)
		for i := range elements {
			elements[i] = "?"
		}
		s := strings.Join(elements, "->")
		return fmt.Sprintf("select id from execution where extra_attrs->%s->>?=?", s)
	}
}
