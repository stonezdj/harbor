package commonevent

import (
	"context"
	"regexp"

	"github.com/goharbor/harbor/src/pkg/notifier/event"
)

type Resolver interface {
	Resolve(*Metadata, *event.Event) error
}

var url2Operation = map[string]Resolver{
	`/api\/v2\.0\/configurations$`:                   configureEventResolver,
	`/c\/login$`:                                     loginEventResolver,
	`/c\/log_out$`:                                   loginEventResolver,
	`/api\/v2\.0\/users$`:                            userResolver,
	`^/api/v2\.0/users/\d+/password$`:                userResolver,
	`^/api/v2\.0/users/\d+/sysadmin$`:                userResolver,
	`^/api/v2\.0/users/\d+$`:                         userResolver,
	`^/api/v2.0/projects/\d+/members`:                projectMemberResolver,
	`^/api/v2.0/projects/\d+/members/\d+$`:           projectMemberResolver,
	`^/api/v2.0/projects$`:                           projectResolver,
	`^/api/v2.0/projects/\d+$`:                       projectResolver,
	`^/api/v2.0/retentions$`:                         tagRetentionResolver,
	`^/api/v2.0/retentions/\d+$`:                     tagRetentionResolver,
	`^/api/v2.0/projects/\d+/immutabletagrules$`:     immutableTagEventResolver,
	`^/api/v2.0/projects/\d+/immutabletagrules/\d+$`: immutableTagEventResolver,
	`^/api/v2.0/system/purgeaudit/schedule$`:         purgeAuditResolver,
	`^/api/v2.0/robots$`:                             robotResolver,
	`^/api/v2.0/robots/\d+$`:                         robotResolver,
}

type Metadata struct {
	Ctx context.Context
	// Username requester username
	Username string
	// RequestPayload http request payload
	RequestPayload string
	// RequestMethod
	RequestMethod string
	// ResponseCode response code
	ResponseCode int
	// RequestURL request URL
	RequestURL string
	// IPAddress IP address of the request
	IPAddress string
	// ResponseLocation response location
	ResponseLocation string
}

// Resolve parse the audit information from CommonEventMetadata
func (c *Metadata) Resolve(event *event.Event) error {
	for url, r := range url2Operation {
		p := regexp.MustCompile(url)
		if p.MatchString(c.RequestURL) {
			return r.Resolve(c, event)
		}
	}
	return nil
}
