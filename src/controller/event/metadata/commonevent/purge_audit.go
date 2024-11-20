package commonevent

import (
	"net/http"
	"time"

	event2 "github.com/goharbor/harbor/src/controller/event"
	"github.com/goharbor/harbor/src/pkg/notifier/event"
)

var purgeAuditResolver = &PurgeAuditEventResolver{}

type PurgeAuditEventResolver struct {
}

func (p *PurgeAuditEventResolver) Resolve(ce *Metadata, event *event.Event) error {
	if ce.RequestMethod != http.MethodPost && ce.RequestMethod != http.MethodDelete && ce.RequestMethod != http.MethodPut {
		return nil
	}
	data := &event2.CommonEvent{}
	data.Operation = "create"
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
