package api

import (
	"errors"
	"fmt"
	"github.com/goharbor/harbor/src/common/models"
	"github.com/goharbor/harbor/src/common/rbac"
	"github.com/goharbor/harbor/src/pkg/immutabletag"
	"strings"
)

// ImmutableTagRuleAPI ...
type ImmutableTagRuleAPI struct {
	BaseController
	manager   immutabletag.RuleManager
	projectID int64
	ID        int64
}

// Prepare validates the user and projectID
func (itr *ImmutableTagRuleAPI) Prepare() {
	itr.BaseController.Prepare()
	if !itr.SecurityCtx.IsAuthenticated() {
		itr.SendUnAuthorizedError(errors.New("Unauthorized"))
		return
	}

	pid, err := itr.GetInt64FromPath(":pid")
	if err != nil || pid <= 0 {
		text := "invalid project ID: "
		if err != nil {
			text += err.Error()
		} else {
			text += fmt.Sprintf("%d", pid)
		}
		itr.SendBadRequestError(errors.New(text))
		return
	}
	itr.projectID = pid

	ruleID, err := itr.GetInt64FromPath(":id")
	if err == nil || ruleID > 0 {
		itr.ID = ruleID
	}

	itr.manager = immutabletag.NewDefaultRuleManager()

	if strings.EqualFold(itr.Ctx.Request.Method, "get") {
		if !itr.requireAccess(rbac.ActionList) {
			return
		}
	} else if strings.EqualFold(itr.Ctx.Request.Method, "put") {
		if !itr.requireAccess(rbac.ActionUpdate) {
			return
		}
	} else if strings.EqualFold(itr.Ctx.Request.Method, "post") {
		if !itr.requireAccess(rbac.ActionCreate) {
			return
		}

	} else if strings.EqualFold(itr.Ctx.Request.Method, "delete") {
		if !itr.requireAccess(rbac.ActionDelete) {
			return
		}
	}
}

func (itr *ImmutableTagRuleAPI) requireAccess(action rbac.Action) bool {
	return itr.RequireProjectAccess(itr.projectID, action, rbac.ResourceImmutableTag)
}

// List list all immutable tag rules of current project
func (itr *ImmutableTagRuleAPI) List() {
	rules, err := itr.manager.QueryImmutableRuleByProjectID(itr.projectID)
	if err != nil {
		itr.SendInternalServerError(err)
	}
	itr.WriteJSONData(rules)
}

// Post create immutable tag rule
func (itr *ImmutableTagRuleAPI) Post() {
	ir := &models.ImmutableRule{}
	if err := itr.DecodeJSONReq(ir); err != nil {
		itr.SendInternalServerError(err)
		return
	}
	if len(ir.TagPattern) == 0 {
		ir.TagPattern = "**"
	}
	if len(ir.RepoPattern) == 0 {
		ir.RepoPattern = "**"
	}
	ir.ProjectID = itr.projectID
	itr.manager.CreateImmutableRule(ir)
}

// Delete delete immutable tag rule
func (itr *ImmutableTagRuleAPI) Delete() {
	if itr.ID <= 0 {
		itr.SendBadRequestError(fmt.Errorf("invalid immutable rule id %d", itr.ID))
	}
	itr.manager.DeleteImmutableRule(itr.ID)
}

// Put update an immutable tag rule
func (itr *ImmutableTagRuleAPI) Put() {
	ir := &models.ImmutableRule{}
	if err := itr.DecodeJSONReq(ir); err != nil {
		itr.SendInternalServerError(err)
		return
	}
	ir.ID = itr.ID
	ir.ProjectID = itr.projectID

	if itr.ID <= 0 {
		itr.SendBadRequestError(fmt.Errorf("invalid immutable rule id %d", itr.ID))
	}
	if len(ir.RepoPattern) == 0 && len(ir.TagPattern) == 0 {
		if _, err := itr.manager.EnableImmutableRule(itr.ID, ir.Enabled); err != nil {
			itr.SendInternalServerError(err)
			return
		}
	} else {
		if _, err := itr.manager.UpdateImmutableRule(itr.ID, ir); err != nil {
			itr.SendInternalServerError(err)
			return
		}
	}

}
