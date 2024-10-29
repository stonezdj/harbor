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
	"strings"
	"time"

	event2 "github.com/goharbor/harbor/src/controller/event"
	"github.com/goharbor/harbor/src/pkg/notifier/event"
)

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
	data := &event2.CommonEvent{}
	if strings.HasSuffix(c.RequestURL, "api/v2.0/configurations") {
		data.Operation = "configuration"
		data.Operator = c.Username
		data.ResourceName = "configuration"
		data.SourceIP = c.IPAddress
		data.Payload = c.RequestPayload
		data.OcurrAt = time.Now()
		data.OperationResult = "success"
		if c.ResponseCode != http.StatusOK {
			data.OperationResult = "failed"
		}
	}
	event.Topic = event2.TopicDeleteArtifact
	event.Data = data
	return nil
}
