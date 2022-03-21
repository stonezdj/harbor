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
	"github.com/goharbor/harbor/src/lib/q"
	"github.com/goharbor/harbor/src/pkg/task"
	taskTesting "github.com/goharbor/harbor/src/testing/pkg/task"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"testing"
)

type ExecutionTestSuite struct {
	suite.Suite
	execMgr *taskTesting.ExecutionManager
	ctl     ExecutionController
}

func (e *ExecutionTestSuite) SetupSuite() {
	e.execMgr = &taskTesting.ExecutionManager{}
	e.ctl = &executionCtl{exeMgr: e.execMgr}
}
func (e *ExecutionTestSuite) TearDownSuite() {
}

func (e *ExecutionTestSuite) TestExecutionCount() {
	e.execMgr.On("Count", mock.Anything, mock.Anything).Return(int64(1), nil)
	count, err := e.ctl.ExecutionCount(nil, purge.VendorType, q.New(q.KeyWords{"VendorType": purge.VendorType}))
	e.Nil(err)
	e.Equal(int64(1), count)
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, &ExecutionTestSuite{})
}

func (e *ExecutionTestSuite) TestGetExecution() {
	e.execMgr.On("List", mock.Anything, mock.Anything).Return([]*task.Execution{
		{
			ID:            1,
			Trigger:       "Manual",
			VendorType:    purge.VendorType,
			StatusMessage: "Success",
		},
	}, nil)

	hs, err := e.ctl.GetExecution(nil, purge.VendorType, int64(1))
	e.Nil(err)
	e.Equal("Manual", hs.Trigger)
}

func (e *ExecutionTestSuite) TestListExecutions() {
	e.execMgr.On("List", mock.Anything, mock.Anything).Return([]*task.Execution{
		{
			ID:      1,
			Trigger: "Manual",
		},
	}, nil)
	hs, err := e.ctl.ListExecutions(nil, purge.VendorType, q.New(q.KeyWords{"VendorType": purge.VendorType}))
	e.Nil(err)
	e.Equal("Manual", hs[0].Trigger)
}
