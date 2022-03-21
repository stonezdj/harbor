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
	"github.com/goharbor/harbor/src/controller/purge"
	"github.com/goharbor/harbor/src/jobservice/job"
	"github.com/goharbor/harbor/src/pkg/task"
	taskTesting "github.com/goharbor/harbor/src/testing/pkg/task"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"testing"
)

type TaskTestSuite struct {
	suite.Suite
	taskMgr *taskTesting.Manager
	ctl     TaskController
}

func (t *TaskTestSuite) SetupSuite() {
	t.taskMgr = &taskTesting.Manager{}
	t.ctl = &taskController{taskMgr: t.taskMgr}
}

func (t *TaskTestSuite) TestListTasks() {
	t.taskMgr.On("List", mock.Anything, mock.Anything).Return([]*task.Task{
		{
			ID:          1,
			ExecutionID: 1,
			Status:      job.RunningStatus.String(),
		},
	}, nil)
	tasks, err := t.ctl.ListTasks(nil, purge.VendorType, nil)
	t.Require().Nil(err)
	t.Require().Len(tasks, 1)
	t.Equal(int64(1), tasks[0].ID)
	t.Equal(int64(1), tasks[0].ExecutionID)
	t.taskMgr.AssertExpectations(t.T())
}

func (t *TaskTestSuite) TestGetTask() {
	t.taskMgr.On("List", mock.Anything, mock.Anything).Return([]*task.Task{
		{
			ID:          1,
			ExecutionID: 1,
			Status:      job.SuccessStatus.String(),
		},
	}, nil)
	tsk, err := t.ctl.GetTask(nil, purge.VendorType, 1)
	t.Nil(err)
	t.Equal(int64(1), tsk.ID)
	t.Equal(int64(1), tsk.ExecutionID)
	t.taskMgr.AssertExpectations(t.T())
}

func (t *TaskTestSuite) TestGetTaskLog() {
	t.taskMgr.On("List", mock.Anything, mock.Anything).Return([]*task.Task{
		{
			ID:          1,
			ExecutionID: 1,
			Status:      job.SuccessStatus.String(),
		},
	}, nil)
	t.taskMgr.On("GetLog", mock.Anything, mock.Anything).Return([]byte("hello world"), nil)

	log, err := t.ctl.GetTaskLog(nil, purge.VendorType, 1)
	t.Nil(err)
	t.Equal([]byte("hello world"), log)
}

func TestTaskTestSuite(t *testing.T) {
	suite.Run(t, &TaskTestSuite{})
}
