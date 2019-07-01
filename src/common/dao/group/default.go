package group

// DefaultGroupPrivilege - dummy group
import (
	"fmt"
	"time"

	"github.com/goharbor/harbor/src/common"
	"github.com/goharbor/harbor/src/common/dao"
	"github.com/goharbor/harbor/src/common/models"
	"github.com/goharbor/harbor/src/common/utils"
	"github.com/goharbor/harbor/src/common/utils/log"
	"github.com/pkg/errors"
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

// AddUserGroup - Add User Group
func AddUserGroup(userGroup models.UserGroup) (int, error) {
	o := dao.GetOrmer()

	sql := "insert into user_group (group_name, group_type, ldap_group_dn, creation_time, update_time) values (?, ?, ?, ?, ?) RETURNING id"
	var id int
	now := time.Now()

	err := o.Raw(sql, userGroup.GroupName, userGroup.GroupType, utils.TrimLower(userGroup.LdapGroupDN), now, now).QueryRow(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// AddHTTPUserGroup - Add HTTP User Group, for http user group, the group_name can not be duplicated
func AddHTTPUserGroup(userGroup models.UserGroup) (int, error) {
	userGroupList, err := QueryHTTPUserGroupByName(userGroup.GroupName)
	if err != nil {
		return 0, err
	}
	if len(userGroupList) > 0 {
		return 0, fmt.Errorf("Duplicated user name, can not create user group %v", userGroup.GroupName)
	}
	return AddHTTPUserGroup(userGroup)
}

// QueryHTTPUserGroupByName - Query HTTP user group by name
func QueryHTTPUserGroupByName(groupName string) ([]*models.UserGroup, error) {
	o := dao.GetOrmer()
	sql := `select id, group_name, group_type, ldap_group_dn from user_group where group_type = 2 and group_name = ? `
	sqlParam := make([]interface{}, 1)
	sqlParam = append(sqlParam, dao.Escape(groupName))
	groups := []*models.UserGroup{}
	_, err := o.Raw(sql, sqlParam).QueryRows(&groups)
	if err != nil {
		return nil, err
	}
	return groups, nil
}

// QueryUserGroup - Query User Group
func QueryUserGroup(query models.UserGroup) ([]*models.UserGroup, error) {
	o := dao.GetOrmer()
	sql := `select id, group_name, group_type, ldap_group_dn from user_group where 1=1 `
	sqlParam := make([]interface{}, 1)
	groups := []*models.UserGroup{}
	if len(query.GroupName) != 0 {
		sql += ` and group_name like ? `
		sqlParam = append(sqlParam, `%`+dao.Escape(query.GroupName)+`%`)
	}

	if query.GroupType != 0 {
		sql += ` and group_type = ? `
		sqlParam = append(sqlParam, query.GroupType)
	}

	if len(query.LdapGroupDN) != 0 {
		sql += ` and ldap_group_dn = ? `
		sqlParam = append(sqlParam, utils.TrimLower(query.LdapGroupDN))
	}
	if query.ID != 0 {
		sql += ` and id = ? `
		sqlParam = append(sqlParam, query.ID)
	}
	_, err := o.Raw(sql, sqlParam).QueryRows(&groups)
	if err != nil {
		return nil, err
	}
	return groups, nil
}

// GetUserGroup ...
func GetUserGroup(id int) (*models.UserGroup, error) {
	userGroup := models.UserGroup{ID: id}
	userGroupList, err := QueryUserGroup(userGroup)
	if err != nil {
		return nil, err
	}
	if len(userGroupList) > 0 {
		return userGroupList[0], nil
	}
	return nil, nil
}

// DeleteUserGroup ...
func DeleteUserGroup(id int) error {
	userGroup := models.UserGroup{ID: id}
	o := dao.GetOrmer()
	_, err := o.Delete(&userGroup)
	if err == nil {
		// Delete all related project members
		sql := `delete from project_member where entity_id = ? and entity_type='g'`
		_, err := o.Raw(sql, id).Exec()
		if err != nil {
			return err
		}
	}
	return err
}

// UpdateUserGroupName ...
func UpdateUserGroupName(id int, groupName string) error {
	userGroup, err := GetUserGroup(id)
	if err != nil {
		return err
	}
	if userGroup.GroupType == common.HTTPGroupType {
		return errors.New("can not update HTTP auth user group")
	}
	log.Debugf("Updating user_group with id:%v, name:%v", id, groupName)
	o := dao.GetOrmer()
	sql := "update user_group set group_name = ? where id =  ? and group_type !=  2 " // http user group name is not allowed to modify
	_, err = o.Raw(sql, groupName, id).Exec()
	return err
}

// OnBoardUserGroup will check if a usergroup exists in usergroup table, if not insert the usergroup and
// put the id in the pointer of usergroup model, if it does exist, return the usergroup's profile.
// This is used for ldap and uaa authentication, such the usergroup can have an ID in Harbor.
// the keyAttribute and combinedKeyAttribute are key columns used to check duplicate usergroup in harbor
func OnBoardUserGroup(g *models.UserGroup, keyAttribute string, combinedKeyAttributes ...string) error {
	g.LdapGroupDN = utils.TrimLower(g.LdapGroupDN)

	o := dao.GetOrmer()
	created, ID, err := o.ReadOrCreate(g, keyAttribute, combinedKeyAttributes...)
	if err != nil {
		return err
	}

	if created {
		g.ID = int(ID)
	} else {
		prevGroup, err := GetUserGroup(int(ID))
		if err != nil {
			return err
		}
		g.ID = prevGroup.ID
		g.GroupName = prevGroup.GroupName
		g.GroupType = prevGroup.GroupType
		g.LdapGroupDN = prevGroup.LdapGroupDN
	}

	return nil
}
