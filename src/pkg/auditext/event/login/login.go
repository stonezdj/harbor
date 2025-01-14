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

package login

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/goharbor/harbor/src/common/rbac"
	event2 "github.com/goharbor/harbor/src/controller/event"
	"github.com/goharbor/harbor/src/controller/event/metadata/commonevent"
	"github.com/goharbor/harbor/src/controller/event/model"
	"github.com/goharbor/harbor/src/lib/config"
	"github.com/goharbor/harbor/src/pkg/notifier/event"
)

func init() {
	var loginResolver = &resolver{}
	commonevent.RegisterResolver(`/c/login$`, loginResolver)
	commonevent.RegisterResolver(`/c/log_out$`, loginResolver)
}

const (
	opLogout       = "logout"
	opLogin        = "login"
	logoutSuffix   = "log_out"
	payloadPattern = `principal=(.*?)&password`
)

type resolver struct {
}

func (l *resolver) Resolve(ce *commonevent.Metadata, event *event.Event) error {
	e := &model.CommonEvent{
		Operator:     ce.Username,
		ResourceType: rbac.ResourceUser.String(),
		ResourceName: ce.Username,
		OcurrAt:      time.Now(),
	}

	// method POST for login, method GET for logout
	if ce.RequestMethod == http.MethodGet {
		e.Operation = opLogout
		e.OperationDescription = opLogout
	} else {
		e.Operation = opLogin
		e.OperationDescription = opLogin
		// Extract the username from payload
		re := regexp.MustCompile(payloadPattern)
		if len(ce.RequestPayload) > 0 {
			match := re.FindStringSubmatch(ce.RequestPayload)
			if len(match) > 1 {
				e.ResourceName = match[1]
				e.Operator = match[1]
			}
		}
	}
	e.OperationResult = true
	if ce.ResponseCode != http.StatusOK {
		e.OperationResult = false
	}
	event.Topic = event2.TopicCommonEvent
	event.Data = e
	return nil
}
func (e *resolver) PreCheck(ctx context.Context, url string, method string) (bool, string) {
	operation := ""
	switch method {
	case http.MethodPost:
		operation = opLogin
	case http.MethodGet:
		operation = opLogout
	}
	if len(operation) == 0 {
		return false, ""
	}
	return config.AuditLogEventEnabled(ctx, fmt.Sprintf("%v_%v", operation, rbac.ResourceUser.String())), ""
}
