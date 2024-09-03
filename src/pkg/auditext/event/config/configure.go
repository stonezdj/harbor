package config

import (
	"context"
	"fmt"
	"net/http"
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
	var configureEventResolver = &ConfigureEventResolver{
		SensitiveAttributes: []string{"ldap_password"},
	}
	commonevent.RegisterResolver(`/api/v2.0/configurations`, configureEventResolver)
}

// ConfigureEventResolver used to resolve the configuration event
type ConfigureEventResolver struct {
	SensitiveAttributes []string
}

func (c *ConfigureEventResolver) Resolve(ce *commonevent.Metadata, evt *event.Event) error {
	data := &model.CommonEvent{}
	data.Operation = "update"
	data.Operator = ce.Username
	data.ResourceName = rbac.ResourceConfiguration.String()
	data.SourceIP = ce.IPAddress
	data.Payload = ext.RedactPayload(ce.RequestPayload, c.SensitiveAttributes)
	data.OcurrAt = time.Now()
	data.OperationDescription = "change configuration"
	data.OperationResult = true
	data.ResourceType = rbac.ResourceConfiguration.String()
	if ce.ResponseCode != http.StatusOK {
		data.OperationResult = false
	}
	evt.Topic = event2.TopicCommonEvent
	evt.Data = data
	return nil
}

func (c *ConfigureEventResolver) PreCheck(ctx context.Context, url string, method string) (bool, string) {
	if method == http.MethodPut {
		return config.AuditLogEnabled(ctx, fmt.Sprintf("%v_%v", "update", rbac.ResourceConfiguration.String())), ""
	}
	return false, ""
}
