package project

import (
	"net/http"
	"strconv"

	"github.com/goharbor/harbor/src/common/rbac"
	"github.com/goharbor/harbor/src/controller/event/metadata/commonevent"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/lib/orm"
	"github.com/goharbor/harbor/src/pkg"
	"github.com/goharbor/harbor/src/pkg/auditext/event"
)

func init() {
	var projectResolver = &ProjectEventResolver{
		EventResolver: event.EventResolver{
			BaseURLPattern:  "/api/v2.0/projects",
			ResourceType:    rbac.ResourceProject.String(),
			SucceedCodes:    []int{http.StatusCreated, http.StatusOK},
			HasResourceName: true,
			IDToNameFunc:    ProjectIDToName,
		},
	}
	commonevent.RegisterResolver(`/api/v2.0/projects$`, projectResolver)
	commonevent.RegisterResolver(`/api/v2.0/projects/\d+$`, projectResolver)
}

type ProjectEventResolver struct {
	event.EventResolver
}

func ProjectIDToName(projectID string) string {
	id, err := strconv.ParseInt(projectID, 10, 32)
	if err != nil {
		log.Errorf("failed to parse projectID: %v to int", projectID)
		return ""
	}
	project, err := pkg.ProjectMgr.Get(orm.Context(), id)
	if err != nil {
		log.Errorf("failed to parse projectID: %v to int, err %v", projectID, err)
		return ""
	}
	return project.Name
}
