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

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/goharbor/harbor/src/jobservice/env"
	"github.com/goharbor/harbor/src/jobservice/hook"
	"github.com/goharbor/harbor/src/jobservice/job"
	"github.com/goharbor/harbor/src/jobservice/job/impl"
	"github.com/goharbor/harbor/src/jobservice/job/impl/replication"
	"github.com/goharbor/harbor/src/jobservice/logger"
	"github.com/goharbor/harbor/src/lib/log"
)

func main() {
	// command line to run the job service
	extralAttr := flag.String("extra_attrs_json", "", "extra attributes for the job")
	id := flag.Int64("id", 0, "job id")
	coreUrl := flag.String("core_url", "http://core", "core url")
	flag.Parse()

	param := job.Parameters{}

	err := json.Unmarshal([]byte(*extralAttr), &param)
	if err != nil {
		fmt.Printf("failed to parse parameter, error %v", err)
	}

	ctx := impl.NewDefaultContext(context.Background())
	ctx.SetLogger(log.New(os.Stdout, log.NewTextFormatter(), log.WarningLevel, 3))

	j := &replication.Replication{}
	if err = j.Run(ctx, param); err != nil {
		fmt.Println(err)
	}

	evt := &hook.Event{
		URL:       fmt.Sprintf("%s/service/notifications/tasks/%d", *coreUrl, *id),
		Timestamp: time.Now().Unix(),
		Data:      &job.StatusChange{Status: job.SuccessStatus.String(), ID: *id, Metadata: &job.StatsInfo{Revision: int64(1000)}},
		Message:   "replication job status changed",
	}

	rootCtx := &env.Context{
		SystemContext: context.Background(),
	}
	hookAgent := hook.NewAgent(rootCtx, "{job_service_namespace}", nil, 3)
	// Hook event sending should not influence the main job flow (because job may call checkin() in the job run).
	if err := hookAgent.Trigger(evt); err != nil {
		logger.Error(err)
	}

}
