package tagretention

import (
	"net/http"

	"github.com/goharbor/harbor/src/common/rbac"
	"github.com/goharbor/harbor/src/controller/event/metadata/commonevent"
	"github.com/goharbor/harbor/src/pkg/auditext/event"
)

func init() {
	var tagRetentionResolver = &TagRetentionEventResolver{
		event.EventResolver{
			BaseURLPattern: "/api/v2.0/retentions",
			ResourceType:   rbac.ResourceTagRetention.String(),
			SucceedCodes:   []int{http.StatusCreated, http.StatusOK},
		},
	}
	commonevent.RegisterResolver(`/api/v2.0/retentions`, tagRetentionResolver)
	commonevent.RegisterResolver(`/api/v2.0/retentions/d+$`, tagRetentionResolver)
}

type TagRetentionEventResolver struct {
	event.EventResolver
}
