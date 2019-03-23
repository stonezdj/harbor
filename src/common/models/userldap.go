package models

// UserLdapTable is the name of table in DB that holds the user ldap info object
const UserLdapTable = "harbor_user_ldap"

// UserLdap - store user ldap information
type UserLdap struct {
	UserID int    `orm:"pk;column(user_id)" json:"user_id"`
	LDAPDN string `orm:"column(ldap_dn)" json:"ldap_dn"`
}

// TableName ...
func (u *UserLdap) TableName() string {
	return UserLdapTable
}
