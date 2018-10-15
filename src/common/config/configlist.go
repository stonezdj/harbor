package config

// Constant for configure item
const (
	//Scope
	UserScope   = "user"
	SystemScope = "system"
	//Group
	LdapBasicGroup = "ldapbasic"
	LdapGroupGroup = "ldapgroup"
	EmailGroup     = "email"
	UAAGroup       = "uaa"
)

var (
	//ConfigList - All configure items used in harbor
	// Steps to onboard a new setting
	// 1. Add configure item in configlist.go
	// 2. Get settings by ClientAPI
	ConfigList = []Item{
		{Scope: UserScope, Group: LdapBasicGroup, EnvironmentKey: "", DefaultValue: "", Name: "ldap_search_base_dn", Type: "string", Editable: true},
		{Scope: UserScope, Group: LdapBasicGroup, EnvironmentKey: "", DefaultValue: "", Name: "ldap_search_scope", Type: "int", Editable: true},
		{Scope: UserScope, Group: LdapBasicGroup, EnvironmentKey: "", DefaultValue: "", Name: "ldap_search", Type: "string", Editable: true},
		{Scope: UserScope, Group: LdapBasicGroup, EnvironmentKey: "", DefaultValue: "", Name: "ldap_search_base_dn", Type: "string", Editable: true},
		{Scope: UserScope, Group: LdapBasicGroup, EnvironmentKey: "", DefaultValue: "", Name: "ldap_search_dn", Type: "string", Editable: true},
	}
)
