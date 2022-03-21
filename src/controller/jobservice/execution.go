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

package jobservice

import (
	"context"
	"github.com/goharbor/harbor/src/lib/errors"
	"github.com/goharbor/harbor/src/lib/q"
	"github.com/goharbor/harbor/src/pkg/task"
)

var (
	// ExecutionCtl ...
	ExecutionCtl = NewExecutionController()
)

// ExecutionController interface to manage execution
type ExecutionController interface {
	// ExecutionCount returns the total count of executions according to the query
	ExecutionCount(ctx context.Context, vendorType string, query *q.Query) (count int64, err error)
	// ListExecutions lists the executions according to the query
	ListExecutions(ctx context.Context, vendorType string, query *q.Query) (executions []*Execution, err error)
	// GetExecution gets the specific execution
	GetExecution(ctx context.Context, vendorType string, executionID int64) (execution *Execution, err error)
}

type executionCtl struct {
	exeMgr task.ExecutionManager
}

// NewExecutionController ...
func NewExecutionController() ExecutionController {
	return &executionCtl{task.ExecMgr}
}

func (e *executionCtl) ExecutionCount(ctx context.Context, vendorType string, query *q.Query) (count int64, err error) {
	query.Keywords["VendorType"] = vendorType
	return e.exeMgr.Count(ctx, query)
}

func (e *executionCtl) ListExecutions(ctx context.Context, vendorType string, query *q.Query) (executions []*Execution, err error) {
	query = q.MustClone(query)
	query.Keywords["VendorType"] = vendorType

	execs, err := e.exeMgr.List(ctx, query)
	if err != nil {
		return nil, err
	}
	var exeResults []*Execution
	for _, exec := range execs {
		exeResults = append(exeResults, convertExecution(exec))
	}
	return exeResults, nil
}

func (e *executionCtl) GetExecution(ctx context.Context, vendorType string, executionID int64) (execution *Execution, err error) {
	execs, err := e.exeMgr.List(ctx, &q.Query{
		Keywords: map[string]interface{}{
			"ID":         executionID,
			"VendorType": vendorType,
		},
	})
	if err != nil {
		return nil, err
	}
	if len(execs) == 0 {
		return nil, errors.New(nil).WithCode(errors.NotFoundCode).
			WithMessage("garbage collection execution %d not found", executionID)
	}
	return convertExecution(execs[0]), nil
}

func convertExecution(exec *task.Execution) *Execution {
	return &Execution{
		ID:            exec.ID,
		Status:        exec.Status,
		StatusMessage: exec.StatusMessage,
		Trigger:       exec.Trigger,
		ExtraAttrs:    exec.ExtraAttrs,
		StartTime:     exec.StartTime,
		EndTime:       exec.EndTime,
	}
}
