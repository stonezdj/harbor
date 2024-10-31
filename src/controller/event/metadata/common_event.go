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

package metadata

import (
	"context"
	"net/http"
	"regexp"
	"time"

	event2 "github.com/goharbor/harbor/src/controller/event"
	"github.com/goharbor/harbor/src/pkg/notifier/event"
)

type funcResolve func(*CommonEventMetadata, *event.Event) error

var url2Operation = map[string]funcResolve{
	`/api\/v2\.0\/configurations$`: ResolveConfigureEvent,
	`/c\/login$`:                   ResolveLoginEvent,
	`/api\/v2\.0\/users$`:          ResolveUserEvent,
}

func ResolveConfigureEvent(ce *CommonEventMetadata, event *event.Event) error {
	data := &event2.CommonEvent{}
	data.Operation = "configuration"
	data.Operator = ce.Username
	data.ResourceName = "configuration"
	data.SourceIP = ce.IPAddress
	data.Payload = ce.RequestPayload
	data.OcurrAt = time.Now()
	data.OperationDescription = "change configuration"
	data.OperationResult = "success"
	if ce.ResponseCode != http.StatusOK {
		data.OperationResult = "failed"
	}
	event.Topic = event2.TopicCommonEvent
	event.Data = data
	return nil
}

func ResolveLoginEvent(ce *CommonEventMetadata, event *event.Event) error {
	if ce.RequestMethod != http.MethodPost {
		return nil
	}
	data := &event2.CommonEvent{}
	data.Operation = "login"
	data.Operator = ce.Username
	data.ResourceName = "user"
	data.SourceIP = ce.IPAddress
	data.Payload = ce.RequestPayload
	data.OcurrAt = time.Now()
	data.OperationDescription = "login"
	data.OperationResult = "success"
	if ce.ResponseCode != http.StatusOK {
		data.OperationResult = "failed"
	}
	event.Topic = event2.TopicCommonEvent
	event.Data = data
	return nil
}

func ResolveUserEvent(ce *CommonEventMetadata, event *event.Event) error {
	if ce.RequestMethod != http.MethodPost && ce.RequestMethod != http.MethodPut {
		return nil
	}
	data := &event2.CommonEvent{}
	data.Operation = "user"
	data.Operator = ce.Username
	data.ResourceName = "user"
	data.SourceIP = ce.IPAddress
	data.Payload = ce.RequestPayload
	data.OcurrAt = time.Now()
	if ce.RequestMethod == http.MethodPost {
		data.OperationDescription = "create user"
	} else {
		data.OperationDescription = "update user"
	}
	data.OperationResult = "success"
	if ce.ResponseCode != http.StatusCreated && ce.ResponseCode != http.StatusOK {
		data.OperationResult = "failed"
	}
	event.Topic = event2.TopicCommonEvent
	event.Data = data
	return nil
}

// CommonEventMetadata used to record an API event related information
type CommonEventMetadata struct {
	Ctx context.Context
	// Username requester username
	Username string
	// RequestPayload http request payload
	RequestPayload string
	// RequestMethod
	RequestMethod string
	// ResponseCode response code
	ResponseCode int
	// RequestURL request URL
	RequestURL string
	// IPAddress IP address of the request
	IPAddress string
}

// Resolve parse the audit information from CommonEventMetadata
func (c *CommonEventMetadata) Resolve(event *event.Event) error {
	for url, resolve := range url2Operation {
		p := regexp.MustCompile(url)
		if p.MatchString(c.RequestURL) {
			return resolve(c, event)
		}
	}
	return nil
}
