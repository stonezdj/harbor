package event

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	event2 "github.com/goharbor/harbor/src/controller/event"
	"github.com/goharbor/harbor/src/controller/event/metadata/commonevent"
	"github.com/goharbor/harbor/src/controller/event/model"
	"github.com/goharbor/harbor/src/lib/config"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/pkg/notifier/event"
)

type ResolveIDToNameFunc func(string) string

type EventResolver struct {
	BaseURLPattern string
	ResourceType   string
	SucceedCodes   []int
	// SensitiveAttributes is the attributes that need to be redacted
	SensitiveAttributes []string
	// HasResourceName indicates if the resource has name, if true, need to resolve the resource name before delete
	HasResourceName bool
	// IDToNameFunc is used to resolve the resource name from resource id
	IDToNameFunc ResolveIDToNameFunc
}

// PreCheck check if the event should be captured and resolve the resource name if needed, if need to resolve the resource name, return the resource name
func (e *EventResolver) PreCheck(ctx context.Context, url string, method string) (capture bool, resourceName string) {
	log.Infof("Capture operation_resource: %v", fmt.Sprintf("%v_%v", e.ResourceType, MethodToOperation(method)))
	capture = config.AuditLogEnabled(ctx, fmt.Sprintf("%v_%v", MethodToOperation(method), e.ResourceType))
	// for delete operation on a resource has name, need to resolve the resource name before delete
	resourceName = ""
	if capture && method == http.MethodDelete && e.HasResourceName {
		re := regexp.MustCompile(fmt.Sprintf(`^%v/(\d+)$`, e.BaseURLPattern))
		m := re.FindStringSubmatch(url)
		if len(m) == 2 && e.IDToNameFunc != nil {
			resourceName = e.IDToNameFunc(m[1])
		}
	}
	return capture, resourceName
}

func (e *EventResolver) Resolve(ce *commonevent.Metadata, event *event.Event) error {
	if ce.RequestMethod != http.MethodPost && ce.RequestMethod != http.MethodDelete && ce.RequestMethod != http.MethodPut {
		return nil
	}
	data := &model.CommonEvent{}
	data.Operator = ce.Username
	data.ResourceType = e.ResourceType
	data.SourceIP = ce.IPAddress
	if ce.RequestMethod != http.MethodDelete && ce.RequestMethod != http.MethodGet {
		data.Payload = RedactPayload(ce.RequestPayload, e.SensitiveAttributes)
	}
	data.OcurrAt = time.Now()
	resourceName := ""
	if ce.RequestMethod == http.MethodPost {
		data.Operation = "create"
		if ce.ResponseLocation != "" {
			// extract resource id from response location
			re := regexp.MustCompile(fmt.Sprintf(`^%v/(\d+)$`, e.BaseURLPattern))
			m := re.FindStringSubmatch(ce.ResponseLocation)
			if len(m) != 2 {
				return nil
			}
			data.ResourceName = m[1]
			if e.IDToNameFunc != nil {
				resourceName = e.IDToNameFunc(m[1])
			}
		}
		if e.HasResourceName && resourceName != "" {
			data.ResourceName = resourceName
		}
		data.OperationDescription = fmt.Sprintf("create %v with name: %v", e.ResourceType, data.ResourceName)
	}
	if ce.RequestMethod == http.MethodDelete {
		re := regexp.MustCompile(fmt.Sprintf(`^%v/(\d+)$`, e.BaseURLPattern))
		m := re.FindStringSubmatch(ce.RequestURL)
		data.Operation = "delete"
		if len(m) != 2 {
			return nil
		}
		data.ResourceName = m[1]
		if e.HasResourceName && ce.ResourceName != "" {
			data.ResourceName = ce.ResourceName
		}
		data.OperationDescription = fmt.Sprintf("delete %v with name: %v", e.ResourceType, data.ResourceName)
	}
	if ce.RequestMethod == http.MethodPut {
		re := regexp.MustCompile(fmt.Sprintf(`^%v/(\d+)`, e.BaseURLPattern))
		m := re.FindStringSubmatch(ce.RequestURL)
		data.Operation = "update"
		if len(m) != 2 {
			return nil
		}
		data.ResourceName = m[1]
		if e.IDToNameFunc != nil {
			resourceName = e.IDToNameFunc(m[1])
		}
		if e.HasResourceName && resourceName != "" {
			data.ResourceName = resourceName
		}

		data.OperationDescription = fmt.Sprintf("update %v with name: %v", e.ResourceType, data.ResourceName)
	}
	data.OperationResult = true
	if !contains(e.SucceedCodes, ce.ResponseCode) {
		data.OperationResult = false
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

// replacePassword recursively replaces "user_password" values with "***"
func replacePassword(data map[string]interface{}, maskAttributes []string) {
	for key, value := range data {
		if inAttributes(key, maskAttributes) {
			data[key] = "***"
		} else if nestedMap, ok := value.(map[string]interface{}); ok {
			replacePassword(nestedMap, maskAttributes)
		} else if nestedArray, ok := value.([]interface{}); ok {
			for _, item := range nestedArray {
				if itemMap, ok := item.(map[string]interface{}); ok {
					replacePassword(itemMap, maskAttributes)
				}
			}
		}
	}
}

func inAttributes(key string, maskAttributes []string) bool {
	for _, attribute := range maskAttributes {
		if key == attribute {
			return true
		}
	}
	return false
}

func RedactPayload(payload string, sensitiveAttributes []string) string {
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &jsonData); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
		return ""
	}
	// Replace user_password values
	replacePassword(jsonData, sensitiveAttributes)
	// Convert the modified map back to JSON
	modifiedJSON, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		log.Fatalf("Error converting to JSON: %v", err)
		return ""
	}
	return string(modifiedJSON)
}

// MethodToOperation converts HTTP method to operation
func MethodToOperation(method string) string {
	switch method {
	case http.MethodPost:
		return "create"
	case http.MethodDelete:
		return "delete"
	case http.MethodPut:
		return "update"
	}
	return ""
}
