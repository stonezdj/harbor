package login

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/goharbor/harbor/src/common/rbac"
	event2 "github.com/goharbor/harbor/src/controller/event"
	"github.com/goharbor/harbor/src/controller/event/metadata/commonevent"
	"github.com/goharbor/harbor/src/controller/event/model"
	"github.com/goharbor/harbor/src/lib/config"
	"github.com/goharbor/harbor/src/pkg/notifier/event"
)

func init() {
	var loginEventResolver = &LoginEventResolver{}
	commonevent.RegisterResolver(`/c/login$`, loginEventResolver)
	commonevent.RegisterResolver(`/c/log_out$`, loginEventResolver)
}

type LoginEventResolver struct {
}

func (l *LoginEventResolver) Resolve(ce *commonevent.Metadata, event *event.Event) error {
	data := &model.CommonEvent{}
	//method POST for login
	//method GET for log_out
	if ce.RequestMethod == http.MethodGet {
		data.Operation = "logout"
	} else {
		data.Operation = "login"
	}
	data.Operator = ce.Username
	data.ResourceType = rbac.ResourceUser.String()
	data.ResourceName = ce.Username
	data.SourceIP = ce.IPAddress
	// Extract the username from payload
	re := regexp.MustCompile(`principal=(.*?)&password`)
	if len(ce.RequestPayload) > 0 {
		match := re.FindStringSubmatch(ce.RequestPayload)
		if len(match) > 1 {
			data.ResourceName = match[1]
			data.Operator = match[1]
		}
	}

	data.OcurrAt = time.Now()
	if strings.HasSuffix(ce.RequestURL, "log_out") {
		data.OperationDescription = "logout"
	} else {
		data.OperationDescription = "login"
	}
	data.OperationResult = true
	if ce.ResponseCode != http.StatusOK {
		data.OperationResult = false
	}
	event.Topic = event2.TopicCommonEvent
	event.Data = data
	return nil
}
func (e *LoginEventResolver) PreCheck(ctx context.Context, url string, method string) (bool, string) {
	operation := ""
	switch method {
	case http.MethodPost:
		operation = "login"
	case http.MethodGet:
		operation = "logout"
	}
	return config.AuditLogEnabled(ctx, fmt.Sprintf("%v_%v", operation, rbac.ResourceUser.String())), ""
}
