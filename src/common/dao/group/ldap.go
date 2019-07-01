// Copyright Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package group

import (
	"github.com/goharbor/harbor/src/common"
	"github.com/goharbor/harbor/src/common/dao"
	"github.com/goharbor/harbor/src/common/models"
	"strings"
)

// GetGroupDNQueryCondition get the part of IN ('XXX', 'XXX') condition
func GetGroupDNQueryCondition(userGroupList []*models.UserGroup) string {
	result := make([]string, 0)
	count := 0
	for _, userGroup := range userGroupList {
		if userGroup.GroupType == common.LdapGroupType {
			result = append(result, "'"+userGroup.LdapGroupDN+"'")
			count++
		}
	}
	// No LDAP Group found
	if count == 0 {
		return ""
	}
	return strings.Join(result, ",")
}

// LDAPGroupContext ...
type LDAPGroupContext []*models.UserGroup

// GetRoleInProject ...
func (c *LDAPGroupContext) GetRoleInProject(projectID int64) ([]int, error) {
	queryCondition := GetGroupDNQueryCondition([]*models.UserGroup(*c))
	return dao.GetRolesByLDAPGroup(projectID, queryCondition)
}

// GetProjects ...
func (c *LDAPGroupContext) GetProjects(query *models.ProjectQueryParam) (total int64, projects []*models.Project, err error) {
	queryCondition := GetGroupDNQueryCondition([]*models.UserGroup(*c))
	count, err := dao.GetTotalGroupProjects(queryCondition, query)
	total = int64(count)
	if err != nil {
		return total, nil, err
	}

	projects, err = dao.GetGroupProjects(queryCondition, query)
	if err != nil {
		return total, nil, err
	}
	return total, projects, nil
}
