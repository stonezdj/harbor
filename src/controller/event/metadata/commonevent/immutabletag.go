package commonevent

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	event2 "github.com/goharbor/harbor/src/controller/event"
	"github.com/goharbor/harbor/src/pkg/notifier/event"
)

var immutableTagEventResolver = &ImmutableTagEventResolver{}

type ImmutableTagEventResolver struct {
}

func (i *ImmutableTagEventResolver) Resolve(ce *Metadata, event *event.Event) error {
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

	data := &event2.CommonEvent{}

	data.Operator = ce.Username
	data.ResourceName = "immutable tag policy"
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
	data.OperationResult = "success"
	if ce.ResponseCode != http.StatusCreated && ce.ResponseCode != http.StatusOK {
		data.OperationResult = "failed"
	}
	event.Topic = event2.TopicCommonEvent
	event.Data = data
	return nil
}
