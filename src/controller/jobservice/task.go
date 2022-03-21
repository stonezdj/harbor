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
	// TaskCtl ...
	TaskCtl = NewTaskController()
)

// TaskController interface to manage task
type TaskController interface {
	// GetTask gets the specific task
	GetTask(ctx context.Context, vendorType string, id int64) (*Task, error)
	// ListTasks lists the tasks according to the query
	ListTasks(ctx context.Context, vendorType string, query *q.Query) (tasks []*Task, err error)
	// GetTaskLog gets log of the specific task
	GetTaskLog(ctx context.Context, vendorType string, id int64) ([]byte, error)
}

type taskController struct {
	taskMgr task.Manager
}

func (t *taskController) GetTask(ctx context.Context, vendorType string, id int64) (*Task, error) {
	tasks, err := t.taskMgr.List(ctx, &q.Query{
		Keywords: map[string]interface{}{
			"ID":         id,
			"VendorType": vendorType,
		},
	})
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, errors.New(nil).WithCode(errors.NotFoundCode).
			WithMessage("vendor type %v, task %d not found", vendorType, id)
	}
	return convertTask(tasks[0]), nil
}

func (t *taskController) ListTasks(ctx context.Context, vendorType string, query *q.Query) (tasks []*Task, err error) {
	query = q.MustClone(query)
	query.Keywords["VendorType"] = vendorType
	tks, err := t.taskMgr.List(ctx, query)
	if err != nil {
		return nil, err
	}
	for _, tk := range tks {
		tasks = append(tasks, convertTask(tk))
	}
	return tasks, nil
}

func convertTask(task *task.Task) *Task {
	return &Task{
		ID:             task.ID,
		ExecutionID:    task.ExecutionID,
		Status:         task.Status,
		StatusMessage:  task.StatusMessage,
		RunCount:       task.RunCount,
		DeleteUntagged: task.GetBoolFromExtraAttrs("delete_untagged"),
		DryRun:         task.GetBoolFromExtraAttrs("dry_run"),
		JobID:          task.JobID,
		CreationTime:   task.CreationTime,
		StartTime:      task.StartTime,
		UpdateTime:     task.UpdateTime,
		EndTime:        task.EndTime,
	}
}

func (t *taskController) GetTaskLog(ctx context.Context, vendorType string, id int64) ([]byte, error) {
	_, err := t.GetTask(ctx, vendorType, id)
	if err != nil {
		return nil, err
	}
	return t.taskMgr.GetLog(ctx, id)
}

// NewTaskController ...
func NewTaskController() TaskController {
	return &taskController{taskMgr: task.Mgr}
}
