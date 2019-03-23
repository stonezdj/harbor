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

package dao

import (
	"errors"
	"fmt"
	"time"

	"github.com/astaxie/beego/orm"

	"github.com/goharbor/harbor/src/common/models"
	"github.com/goharbor/harbor/src/common/utils"

	"github.com/goharbor/harbor/src/common/utils/log"
)

// GetUser ...
func GetUser(query models.User) (*models.User, error) {

	o := GetOrmer()

	sql := `select user_id, username, password, email, realname, comment, reset_uuid, salt,
		sysadmin_flag, creation_time, update_time
		from harbor_user u
		where deleted = false `
	queryParam := make([]interface{}, 1)
	if query.UserID != 0 {
		sql += ` and user_id = ? `
		queryParam = append(queryParam, query.UserID)
	}

	if query.Username != "" {
		sql += ` and username = ? `
		queryParam = append(queryParam, query.Username)
	}

	if query.ResetUUID != "" {
		sql += ` and reset_uuid = ? `
		queryParam = append(queryParam, query.ResetUUID)
	}

	if query.Email != "" {
		sql += ` and email = ? `
		queryParam = append(queryParam, query.Email)
	}

	var u []models.User
	n, err := o.Raw(sql, queryParam).QueryRows(&u)

	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}

	if n > 1 {
		return nil, fmt.Errorf("got more than one user when executing: %s param: %v", sql, queryParam)
	}

	return &u[0], nil
}

// LoginByDb is used for user to login with database auth mode.
func LoginByDb(auth models.AuthModel) (*models.User, error) {
	o := GetOrmer()

	var users []models.User
	n, err := o.Raw(`select * from harbor_user where (username = ? or email = ?) and deleted = false`,
		auth.Principal, auth.Principal).QueryRows(&users)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}

	user := users[0]

	if user.Password != utils.Encrypt(auth.Password, user.Salt) {
		return nil, nil
	}

	user.Password = "" // do not return the password

	return &user, nil
}

// GetTotalOfUsers ...
func GetTotalOfUsers(query *models.UserQuery) (int64, error) {
	return userQueryConditions(query).Count()
}

// ListUsers lists all users according to different conditions.
func ListUsers(query *models.UserQuery) ([]models.User, error) {
	users := []models.User{}
	_, err := userQueryConditions(query).Limit(-1).
		OrderBy("username").
		All(&users)
	return users, err
}

// FetchUserLDAPInfo Fetch User LDAP information and attach to user model
func FetchUserLDAPInfo(users []models.User) error {

	if len(users) == 0 {
		return nil
	}
	var userIDs []interface{}
	for _, u := range users {
		userIDs = append(userIDs, u.UserID)
	}

	// fetch user ldap information
	var userLdaps []models.UserLdap
	userIDLdapMap := map[int]string{}
	qs := GetOrmer().QueryTable(models.UserLdapTable)

	qs.Filter("user_id__in", userIDs...)
	_, err := qs.All(&userLdaps)
	if err != nil {
		return err
	}
	for _, userLdap := range userLdaps {
		userIDLdapMap[userLdap.UserID] = userLdap.LDAPDN
	}

	for i := range users {
		if _, ok := userIDLdapMap[users[i].UserID]; ok {
			users[i].LDAPDN = userIDLdapMap[users[i].UserID]
		}
	}
	return nil
}

func userQueryConditions(query *models.UserQuery) orm.QuerySeter {
	qs := GetOrmer().QueryTable(&models.User{}).
		Filter("deleted", 0).
		Filter("user_id__gt", 1)

	if query == nil {
		return qs
	}

	if len(query.Username) > 0 {
		qs = qs.Filter("username__contains", query.Username)
	}

	if len(query.Email) > 0 {
		qs = qs.Filter("email__contains", query.Email)
	}

	return qs
}

// ToggleUserAdminRole gives a user admin role.
func ToggleUserAdminRole(userID int, hasAdmin bool) error {
	o := GetOrmer()
	queryParams := make([]interface{}, 1)
	sql := `update harbor_user set sysadmin_flag = ? where user_id = ?`
	queryParams = append(queryParams, hasAdmin)
	queryParams = append(queryParams, userID)
	r, err := o.Raw(sql, queryParams).Exec()
	if err != nil {
		return err
	}

	if _, err := r.RowsAffected(); err != nil {
		return err
	}

	return nil
}

// ChangeUserPassword ...
func ChangeUserPassword(u models.User) error {
	u.UpdateTime = time.Now()
	u.Salt = utils.GenerateRandomString()
	u.Password = utils.Encrypt(u.Password, u.Salt)
	_, err := GetOrmer().Update(&u, "Password", "Salt", "UpdateTime")
	return err
}

// ResetUserPassword ...
func ResetUserPassword(u models.User) error {
	o := GetOrmer()
	r, err := o.Raw(`update harbor_user set password=?, reset_uuid=? where reset_uuid=?`, utils.Encrypt(u.Password, u.Salt), "", u.ResetUUID).Exec()
	if err != nil {
		return err
	}
	count, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("no record be changed, reset password failed")
	}
	return nil
}

// UpdateUserResetUUID ...
func UpdateUserResetUUID(u models.User) error {
	o := GetOrmer()
	_, err := o.Raw(`update harbor_user set reset_uuid=? where email=?`, u.ResetUUID, u.Email).Exec()
	return err
}

// DeleteUser ...
func DeleteUser(userID int) error {
	o := GetOrmer()

	user, err := GetUser(models.User{
		UserID: userID,
	})
	if err != nil {
		return err
	}

	name := fmt.Sprintf("%s#%d", user.Username, user.UserID)
	email := fmt.Sprintf("%s#%d", user.Email, user.UserID)

	_, err = o.Raw(`update harbor_user 
		set deleted = true, username = ?, email = ?
		where user_id = ?`, name, email, userID).Exec()
	if err != nil {
		return err
	}
	_, err = o.Raw("delete from harbor_user_ldap where user_id = ? ", userID).Exec()
	return err
}

// ChangeUserProfile - Update user in local db,
// cols to specify the columns need to update,
// Email, and RealName, Comment are updated by default.
func ChangeUserProfile(user models.User, cols ...string) error {
	o := GetOrmer()
	if len(cols) == 0 {
		cols = []string{"Email", "Realname", "Comment"}
	}
	if _, err := o.Update(&user, cols...); err != nil {
		log.Errorf("update user failed, error: %v", err)
		return err
	}
	return nil
}

// OnBoardUser will check if a user exists in user table, if not insert the user and
// put the id in the pointer of user model, if it does exist, return the user's profile.
// This is used for ldap and uaa authentication, such the user can have an ID in Harbor.
func OnBoardUser(u *models.User) error {
	o := GetOrmer()
	o.Begin()
	HasAdminRole := u.HasAdminRole
	// Not set has_admin_role when OnBoardUser
	u.HasAdminRole = false
	userLdap := &models.UserLdap{
		LDAPDN: u.LDAPDN,
	}
	created, id, err := o.ReadOrCreate(u, "Username")

	if err != nil {
		o.Rollback()
		return err
	}
	if created {
		u.UserID = int(id)
	} else {
		existing, err := GetUser(*u)
		if err != nil {
			o.Rollback()
			return err
		}
		u.Email = existing.Email
		u.HasAdminRole = existing.HasAdminRole
		u.Realname = existing.Realname
		u.UserID = existing.UserID
	}
	userLdap.UserID = u.UserID
	if len(userLdap.LDAPDN) > 0 {
		_, err = o.InsertOrUpdate(userLdap, "user_id")
		if err != nil {
			o.Rollback()
			return err
		}
	}

	u.HasAdminRole = HasAdminRole
	o.Commit()
	return nil
}

// IsSuperUser checks if the user is super user(conventionally id == 1) of Harbor
func IsSuperUser(username string) bool {
	u, err := GetUser(models.User{
		Username: username,
	})
	log.Debugf("Check if user %s is super user", username)
	if err != nil {
		log.Errorf("Failed to get user from DB, username: %s, error: %v", username, err)
		return false
	}
	return u != nil && u.UserID == 1
}

// CleanUser - Clean this user information from DB
func CleanUser(id int64) error {
	if _, err := GetOrmer().QueryTable(&models.User{}).
		Filter("UserID", id).Delete(); err != nil {
		return err
	}
	return nil
}

// CleanUserLdap - clean user ldap info
func CleanUserLdap(id int64) error {
	if _, err := GetOrmer().QueryTable(&models.UserLdap{}).
		Filter("UserID", id).Delete(); err != nil {
		return err
	}
	return nil
}
