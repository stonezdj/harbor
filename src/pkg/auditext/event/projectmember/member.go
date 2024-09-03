package projectmember

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
	var projectMemberResolver = &ProjectMemberEventResolver{}
	commonevent.RegisterResolver(`^/api/v2.0/projects/\d+/members`, projectMemberResolver)
	commonevent.RegisterResolver(`^/api/v2.0/projects/\d+/members/\d+`, projectMemberResolver)
}

type ProjectMemberEventResolver struct {
}

func (p *ProjectMemberEventResolver) Resolve(ce *commonevent.Metadata, event *event.Event) error {
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

	data := &model.CommonEvent{}
	data.Operator = ce.Username
	data.ResourceType = rbac.ResourceMember.String()
	data.SourceIP = ce.IPAddress
	data.Payload = ce.RequestPayload
	data.OcurrAt = time.Now()
	if ce.RequestMethod == http.MethodPost {
		data.Operation = "create"
		data.OperationDescription = fmt.Sprintf("add project member to project with project id %v", projectID)
	} else if ce.RequestMethod == http.MethodDelete {
		data.Operation = "delete"
		data.OperationDescription = fmt.Sprintf("delete project member from project with project id %v, member id: %v", projectID, memberID)
	} else {
		data.Operation = "update"
		data.OperationDescription = fmt.Sprintf("update project member to project %v with project id %v", projectID, memberID)
	}
	data.OperationResult = true
	if ce.ResponseCode != http.StatusCreated && ce.ResponseCode != http.StatusOK {
		data.OperationResult = false
	}
	event.Topic = event2.TopicCommonEvent
	event.Data = data
	return nil
}

func (p *ProjectMemberEventResolver) PreCheck(ctx context.Context, url string, method string) (bool, string) {
	return config.AuditLogEnabled(ctx, fmt.Sprintf("%v_%v", ext.MethodToOperation(method), rbac.ResourceMember.String())), ""
}
