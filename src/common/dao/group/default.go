package group

// DefaultGroupPrivilege - dummy group
import (
	"github.com/goharbor/harbor/src/common"
	"github.com/goharbor/harbor/src/common/dao"
	"github.com/goharbor/harbor/src/common/models"
)

// PrivilegeInterface - The interface to implement group specific privileges
// If need to add another group support, implement this interface for this group and add support in GetPrivilegeInterface factory method
type PrivilegeInterface interface {
	// GetRoleInProject - Get role in current project. the role only take effective when user role in this project is not defined.
	GetRoleInProject(projectID int64) ([]int, error)
	// GetProjects - Get the visiable project in current group context
	GetProjects(query *models.ProjectQueryParam) (total int64, projects []*models.Project, err error)
}

// DefaultGroupPrivilege - dummy group privilege implementation
type DefaultGroupPrivilege struct {
}

// GetRoleInProject - dummy group return empty role.
func (c *DefaultGroupPrivilege) GetRoleInProject(projectID int64) ([]int, error) {
	return []int{}, nil
}

// GetProjects - dummy group fetch all projects by current user
func (c *DefaultGroupPrivilege) GetProjects(query *models.ProjectQueryParam) (total int64, projects []*models.Project, err error) {
	total, err = dao.GetTotalOfProjects(query)
	if err != nil {
		return total, nil, err
	}
	projects, err = dao.GetProjects(query)
	if err != nil {
		return total, nil, err
	}
	return total, projects, nil
}

// CreatePrivilegeInterface - The factory method for create PrivilegeInterface
func CreatePrivilegeInterface(gc *models.GroupContext) PrivilegeInterface {
	if gc.GroupType == common.LDAPGroupType {
		ldapGroupContext := LDAPGroupContext(gc.UserGroup)
		return &ldapGroupContext
	}
	return &DefaultGroupPrivilege{}
}
