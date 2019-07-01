package group

import (
	"strings"

	"github.com/goharbor/harbor/src/common"
	"github.com/goharbor/harbor/src/common/dao"
	"github.com/goharbor/harbor/src/common/models"
)

// HTTPGroupContext ...
type HTTPGroupContext []*models.UserGroup

// GetHTTPGroupQueryCondition get the part of IN ('XXX', 'XXX') condition
func GetHTTPGroupQueryCondition(userGroupList []*models.UserGroup) string {
	result := make([]string, 0)
	count := 0
	for _, userGroup := range userGroupList {
		if userGroup.GroupType == common.HTTPGroupType {
			result = append(result, "'"+userGroup.GroupName+"'")
			count++
		}
	}
	// No HTTP Group found
	if count == 0 {
		return ""
	}
	return strings.Join(result, ",")
}

// GetRoleInProject ...
func (c *HTTPGroupContext) GetRoleInProject(projectID int64) ([]int, error) {
	queryCondition := GetHTTPGroupQueryCondition([]*models.UserGroup(*c))
	return dao.GetRolesByHTTPGroup(projectID, queryCondition)
}

// GetProjects ...
func (c *HTTPGroupContext) GetProjects(query *models.ProjectQueryParam) (total int64, projects []*models.Project, err error) {
	queryCondition := GetHTTPGroupQueryCondition([]*models.UserGroup(*c))
	count, err := dao.GetTotalGroupProjects(queryCondition, query)
	total = int64(count)
	if err != nil {
		return total, nil, err
	}

	projects, err = dao.GetHTTPGroupProjects(queryCondition, query)
	if err != nil {
		return total, nil, err
	}
	return total, projects, nil
}
