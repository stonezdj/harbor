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
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	event2 "github.com/goharbor/harbor/src/controller/event"
	"github.com/goharbor/harbor/src/pkg/notifier/event"
)

type funcResolve func(*CommonEventMetadata, *event.Event) error

var url2Operation = map[string]funcResolve{
	`/api\/v2\.0\/configurations$`:                   ResolveConfigureEvent,
	`/c\/login$`:                                     ResolveLoginEvent,
	`/c\/log_out$`:                                   ResolveLoginEvent,
	`/api\/v2\.0\/users$`:                            ResolveUserEvent,
	`^/api/v2\.0/users/\d+/password$`:                ResolveUserEvent,
	`^/api/v2\.0/users/\d+/sysadmin$`:                ResolveUserEvent,
	`^/api/v2\.0/users/\d+$`:                         ResolveUserEvent,
	`^/api/v2.0/projects/\d+/members`:                ResolveProjectMemberEvent,
	`^/api/v2.0/projects/\d+/members/\d+$`:           ResolveProjectMemberEvent,
	`^/api/v2.0/projects$`:                           ResolveProjectEvent,
	`^/api/v2.0/projects/\d+$`:                       ResolveProjectEvent,
	`^/api/v2.0/retentions$`:                         ResolveTagRetentionEvent,
	`^/api/v2.0/retentions/\d+$`:                     ResolveTagRetentionEvent,
	`^/api/v2.0/projects/\d+/immutabletagrules$`:     ResolveImmutableTagEvent,
	`^/api/v2.0/projects/\d+/immutabletagrules/\d+$`: ResolveImmutableTagEvent,
	`^/api/v2.0/system/purgeaudit/schedule$`:         ResolvePurgeAuditEvent,
	`^/api/v2.0/robots$`:                             ResolveRobotAccountEvent,
	`^/api/v2.0/robots/\d+$`:                         ResolveRobotAccountEvent,
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

	data := &event2.CommonEvent{}
	//method POST for login
	//method GET for log_out
	if ce.RequestMethod == http.MethodGet {
		data.Operation = "logout"
	} else {
		data.Operation = "login"
	}
	data.Operator = ce.Username
	data.ResourceType = "user"
	data.ResourceName = ce.Username
	data.SourceIP = ce.IPAddress
	data.Payload = ce.RequestPayload
	data.OcurrAt = time.Now()
	if strings.HasSuffix(ce.RequestURL, "log_out") {
		data.OperationDescription = "logout"
	} else {
		data.OperationDescription = "login"
	}
	data.OperationResult = "success"
	if ce.ResponseCode != http.StatusOK {
		data.OperationResult = "failed"
	}
	event.Topic = event2.TopicCommonEvent
	event.Data = data
	return nil
}

func ResolveUserEvent(ce *CommonEventMetadata, event *event.Event) error {
	userResolver := &EventResolver{
		BaseURLPattern: "/api/v2.0/users",
		ResourceType:   "user",
		SucceedCodes:   []int{http.StatusCreated, http.StatusOK},
	}
	return userResolver.Resolve(ce, event)
}

func ResolveProjectEvent(ce *CommonEventMetadata, event *event.Event) error {
	projectResolver := &EventResolver{
		BaseURLPattern: "/api/v2.0/projects",
		ResourceType:   "project",
		SucceedCodes:   []int{http.StatusCreated, http.StatusOK},
	}
	return projectResolver.Resolve(ce, event)
}

func ResolveProjectMemberEvent(ce *CommonEventMetadata, event *event.Event) error {
	if ce.RequestMethod != http.MethodPost && ce.RequestMethod != http.MethodDelete && ce.RequestMethod != http.MethodPut {
		return nil
	}
	re := regexp.MustCompile(`^/api/v2\.0/projects/(\d+)`)
	matches := re.FindStringSubmatch(ce.RequestURL)
	projectID := ""
	if len(matches) >= 2 {
		projectID = matches[1]
	}

	re2 := regexp.MustCompile(`^/api/v2\.0/projects/\d+/members/(\d+)$`)
	matches2 := re2.FindStringSubmatch(ce.RequestURL)
	memberID := ""
	if len(matches2) >= 2 {
		memberID = matches2[1]
	}

	data := &event2.CommonEvent{}
	data.Operation = "project member"
	data.Operator = ce.Username
	data.ResourceType = "project member"
	data.SourceIP = ce.IPAddress
	data.Payload = ce.RequestPayload
	data.OcurrAt = time.Now()
	if ce.RequestMethod == http.MethodPost {
		data.OperationDescription = fmt.Sprintf("add project member to project with project id %v", projectID)
	} else if ce.RequestMethod == http.MethodDelete {
		data.OperationDescription = fmt.Sprintf("delete project member from project with project id %v, member id: %v", projectID, memberID)
	} else {
		data.OperationDescription = fmt.Sprintf("update project member to project %v with project id %v", projectID, memberID)
	}
	data.OperationResult = "success"
	if ce.ResponseCode != http.StatusCreated && ce.ResponseCode != http.StatusOK {
		data.OperationResult = "failed"
	}
	event.Topic = event2.TopicCommonEvent
	event.Data = data
	return nil
}

func ResolveTagRetentionEvent(ce *CommonEventMetadata, event *event.Event) error {
	tagRetentionResolver := &EventResolver{
		BaseURLPattern: "/api/v2.0/retentions",
		ResourceType:   "tag retention policy",
		SucceedCodes:   []int{http.StatusCreated, http.StatusOK},
	}
	return tagRetentionResolver.Resolve(ce, event)
}

func ResolveImmutableTagEvent(ce *CommonEventMetadata, event *event.Event) error {
	if ce.RequestMethod != http.MethodPost && ce.RequestMethod != http.MethodDelete && ce.RequestMethod != http.MethodPut {
		return nil
	}
	re := regexp.MustCompile(`^/api/v2\.0/projects/(\d+)`)
	matches := re.FindStringSubmatch(ce.RequestURL)
	projectID := ""
	if len(matches) >= 2 {
		projectID = matches[1]
	}

	re2 := regexp.MustCompile(`^/api/v2\.0/projects/\d+/immutabletagrules/(\d+)$`)
	matches2 := re2.FindStringSubmatch(ce.RequestURL)
	immutableTagID := ""
	if len(matches2) >= 2 {
		immutableTagID = matches2[1]
	}

	data := &event2.CommonEvent{}
	data.Operation = "immutable tag"
	data.Operator = ce.Username
	data.ResourceName = "immutable tag policy"
	data.SourceIP = ce.IPAddress
	data.Payload = ce.RequestPayload
	data.OcurrAt = time.Now()
	if ce.RequestMethod == http.MethodPost {
		data.OperationDescription = fmt.Sprintf("add immutable tag to project with project id %v", projectID)
	} else if ce.RequestMethod == http.MethodDelete {
		data.OperationDescription = fmt.Sprintf("delete immutable tag from project with project id %v, immutable tag id: %v", projectID, immutableTagID)
	} else {
		data.OperationDescription = fmt.Sprintf("update immutable tag to project %v with project id %v", projectID, immutableTagID)
	}
	data.OperationResult = "success"
	if ce.ResponseCode != http.StatusCreated && ce.ResponseCode != http.StatusOK {
		data.OperationResult = "failed"
	}
	event.Topic = event2.TopicCommonEvent
	event.Data = data
	return nil
}

func ResolvePurgeAuditEvent(ce *CommonEventMetadata, event *event.Event) error {
	if ce.RequestMethod != http.MethodPost && ce.RequestMethod != http.MethodDelete && ce.RequestMethod != http.MethodPut {
		return nil
	}
	data := &event2.CommonEvent{}
	data.Operation = "purge audit"
	data.Operator = ce.Username
	data.ResourceName = "purge audit"
	data.SourceIP = ce.IPAddress
	data.Payload = ce.RequestPayload
	data.OcurrAt = time.Now()
	if ce.RequestMethod == http.MethodPost {
		data.OperationDescription = "create purge audit"
	}
	if ce.RequestMethod == http.MethodDelete {
		data.OperationDescription = "delete purge audit"
	}
	if ce.RequestMethod == http.MethodPut {
		data.OperationDescription = "update purge audit"
	}
	data.OperationResult = "success"
	if ce.ResponseCode != http.StatusCreated && ce.ResponseCode != http.StatusOK {
		data.OperationResult = "failed"
	}
	event.Topic = event2.TopicCommonEvent
	event.Data = data
	return nil
}

func ResolveRobotAccountEvent(ce *CommonEventMetadata, event *event.Event) error {
	robotResolver := &EventResolver{
		BaseURLPattern: "/api/v2.0/robots",
		ResourceType:   "robot",
		SucceedCodes:   []int{http.StatusCreated, http.StatusOK},
	}
	return robotResolver.Resolve(ce, event)
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
	// ResponseLocation response location
	ResponseLocation string
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
