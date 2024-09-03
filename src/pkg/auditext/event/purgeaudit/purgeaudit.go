package purgeaudit

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
	var purgeAuditResolver = &PurgeAuditEventResolver{}
	commonevent.RegisterResolver(`^/api/v2.0/system/purgeaudit/schedule$`, purgeAuditResolver)
}

type PurgeAuditEventResolver struct {
}

func (p *PurgeAuditEventResolver) Resolve(ce *commonevent.Metadata, event *event.Event) error {
	if ce.RequestMethod != http.MethodPost && ce.RequestMethod != http.MethodDelete && ce.RequestMethod != http.MethodPut {
		return nil
	}
	data := &model.CommonEvent{}
	data.Operation = "create"
	data.Operator = ce.Username
	data.ResourceName = rbac.ResourcePurgeAuditLog.String()
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
	data.OperationResult = true
	if ce.ResponseCode != http.StatusCreated && ce.ResponseCode != http.StatusOK {
		data.OperationResult = false
	}
	event.Topic = event2.TopicCommonEvent
	event.Data = data
	return nil
}
func (p *PurgeAuditEventResolver) PreCheck(ctx context.Context, url string, method string) (bool, string) {
	return config.AuditLogEnabled(ctx, fmt.Sprintf("%v_%v", ext.MethodToOperation(method), rbac.ResourcePurgeAuditLog.String())), ""
}
