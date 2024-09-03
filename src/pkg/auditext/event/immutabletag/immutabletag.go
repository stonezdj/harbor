package immutabletag

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
	ext "github.com/goharbor/harbor/src/pkg/auditext/event"
	"github.com/goharbor/harbor/src/pkg/notifier/event"
)

func init() {
	var immutableTagEventResolver = &ImmutableTagEventResolver{}
	commonevent.RegisterResolver(`^/api/v2.0/projects/\d+/immutabletagrules$`, immutableTagEventResolver)
	commonevent.RegisterResolver(`^/api/v2.0/projects/\d+/immutabletagrules/\d+$`, immutableTagEventResolver)
}

var immutableTagEventResolver = &ImmutableTagEventResolver{}

var immutableTag = rbac.ResourceImmutableTag.String()

type ImmutableTagEventResolver struct {
}

func (i *ImmutableTagEventResolver) Resolve(ce *commonevent.Metadata, evt *event.Event) error {
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

	data := &model.CommonEvent{}

	data.Operator = ce.Username
	data.ResourceName = immutableTag
	data.ResourceType = immutableTag
	data.SourceIP = ce.IPAddress
	data.Payload = ce.RequestPayload
	data.OcurrAt = time.Now()
	if ce.RequestMethod == http.MethodPost {
		data.Operation = "create"
		data.OperationDescription = fmt.Sprintf("add immutable tag to project with project id %v", projectID)
	} else if ce.RequestMethod == http.MethodDelete {
		data.Operation = "delete"
		data.OperationDescription = fmt.Sprintf("delete immutable tag from project with project id %v, immutable tag id: %v", projectID, immutableTagID)
	} else {
		data.Operation = "update"
		data.OperationDescription = fmt.Sprintf("update immutable tag to project %v with project id %v", projectID, immutableTagID)
	}
	data.OperationResult = true
	if ce.ResponseCode != http.StatusCreated && ce.ResponseCode != http.StatusOK {
		data.OperationResult = false
	}
	evt.Topic = event2.TopicCommonEvent
	evt.Data = data
	return nil
}

func (e *ImmutableTagEventResolver) PreCheck(ctx context.Context, url string, method string) (bool, string) {
	return config.AuditLogEnabled(ctx, fmt.Sprintf("%v_%v", ext.MethodToOperation(method), immutableTag)), ""
}
