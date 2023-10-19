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

	"github.com/goharbor/harbor/src/jobservice/job"
	"github.com/goharbor/harbor/src/jobservice/job/impl"
	"github.com/goharbor/harbor/src/jobservice/job/impl/replication"
	"github.com/goharbor/harbor/src/lib/log"
)

func main() {
	// command line to run the job service
	extralAttr := flag.String("extra_attrs_json", "", "extra attributes for the job")
	flag.Parse()

	param := job.Parameters{}

	err := json.Unmarshal([]byte(*extralAttr), &param)
	if err != nil {
		fmt.Printf("failed to parse parameter, error %v", err)
	}

	ctx := impl.NewDefaultContext(context.Background())
	ctx.SetLogger(log.New(os.Stdout, log.NewTextFormatter(), log.WarningLevel, 3))

	job := &replication.Replication{}
	if err = job.Run(ctx, param); err != nil {
		fmt.Println(err)
	}

}
