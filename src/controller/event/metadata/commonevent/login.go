package commonevent

import (
	"net/http"
	"strings"
	"time"

	event2 "github.com/goharbor/harbor/src/controller/event"
	"github.com/goharbor/harbor/src/pkg/notifier/event"
)

var loginEventResolver = &LoginEventResolver{}

type LoginEventResolver struct {
}

func (l *LoginEventResolver) Resolve(ce *Metadata, event *event.Event) error {
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
