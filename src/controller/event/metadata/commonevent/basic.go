package commonevent

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	event2 "github.com/goharbor/harbor/src/controller/event"
	"github.com/goharbor/harbor/src/pkg/notifier/event"
)

var userResolver = &EventResolver{
	BaseURLPattern:      "/api/v2.0/users",
	ResourceType:        "user",
	SucceedCodes:        []int{http.StatusCreated, http.StatusOK},
	SensitiveAttributes: []string{"password"},
}

var projectResolver = &EventResolver{
	BaseURLPattern: "/api/v2.0/projects",
	ResourceType:   "project",
	SucceedCodes:   []int{http.StatusCreated, http.StatusOK},
}

var tagRetentionResolver = &EventResolver{
	BaseURLPattern: "/api/v2.0/retentions",
	ResourceType:   "tag retention policy",
	SucceedCodes:   []int{http.StatusCreated, http.StatusOK},
}

var robotResolver = &EventResolver{
	BaseURLPattern: "/api/v2.0/robots",
	ResourceType:   "robot",
	SucceedCodes:   []int{http.StatusCreated, http.StatusOK},
}

type EventResolver struct {
	BaseURLPattern      string
	ResourceType        string
	SucceedCodes        []int
	SensitiveAttributes []string
}

func (e *EventResolver) Resolve(ce *Metadata, event *event.Event) error {
	if ce.RequestMethod != http.MethodPost && ce.RequestMethod != http.MethodDelete && ce.RequestMethod != http.MethodPut {
		return nil
	}
	data := &event2.CommonEvent{}
	data.Operator = ce.Username
	data.ResourceType = e.ResourceType
	data.SourceIP = ce.IPAddress
	data.Payload = RedactPayload(ce.RequestPayload, e.SensitiveAttributes)
	data.OcurrAt = time.Now()
	if ce.RequestMethod == http.MethodPost {
		data.Operation = "create"
		data.OperationDescription = fmt.Sprintf("create %v", e.ResourceType)
		if ce.ResponseLocation != "" {
			// extract resource id from response location
			re := regexp.MustCompile(fmt.Sprintf(`^%v/(\d+)$`, e.BaseURLPattern))
			m := re.FindStringSubmatch(ce.ResponseLocation)
			if len(m) != 2 {
				return nil
			}
			data.ResourceName = m[1]
		}
	}
	if ce.RequestMethod == http.MethodDelete {
		re := regexp.MustCompile(fmt.Sprintf(`^%v/(\d+)$`, e.BaseURLPattern))
		m := re.FindStringSubmatch(ce.RequestURL)
		data.Operation = "delete"
		if len(m) != 2 {
			return nil
		}
		data.ResourceName = m[1]
		data.OperationDescription = fmt.Sprintf("delete %v with %v id %v", e.ResourceType, e.ResourceType, data.ResourceName)
	}
	if ce.RequestMethod == http.MethodPut {
		re := regexp.MustCompile(fmt.Sprintf(`^%v/(\d+)`, e.BaseURLPattern))
		m := re.FindStringSubmatch(ce.RequestURL)
		data.Operation = "update"
		if len(m) != 2 {
			return nil
		}
		resourceID := m[1]
		data.OperationDescription = fmt.Sprintf("update %v with %v id %v", e.ResourceType, e.ResourceType, resourceID)
	}
	data.OperationResult = "success"
	if !contains(e.SucceedCodes, ce.ResponseCode) {
		data.OperationResult = "failed"
	}
	event.Topic = event2.TopicCommonEvent
	event.Data = data
	return nil
}

func contains(slice []int, value int) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
