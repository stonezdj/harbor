package commonevent

import (
	"net/http"
	"time"

	event2 "github.com/goharbor/harbor/src/controller/event"
	"github.com/goharbor/harbor/src/pkg/notifier/event"
)

var configureEventResolver = &ConfigureEventResolver{
	SensitiveAttributes: []string{"ldap_password"},
}

// ConfigureEventResolver used to resolve the configuration event
type ConfigureEventResolver struct {
	SensitiveAttributes []string
}

func (c *ConfigureEventResolver) Resolve(ce *Metadata, event *event.Event) error {
	data := &event2.CommonEvent{}
	data.Operation = "configuration"
	data.Operator = ce.Username
	data.ResourceName = "configuration"
	data.SourceIP = ce.IPAddress
	data.Payload = RedactPayload(ce.RequestPayload, c.SensitiveAttributes)
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
