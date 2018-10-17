package config


var (
	//ConfigList - All configure items used in harbor
	// Steps to onboard a new setting
	// 1. Add configure item in configlist.go
	// 2. Get settings by ClientAPI
	ConfigList = []Item{
		{Scope: UserScope, Group: LdapBasicGroup, EnvironmentKey: "", DefaultValue: "", Name: "ldap_search_base_dn", Type: StringType, Editable: true},
		{Scope: UserScope, Group: LdapBasicGroup, EnvironmentKey: "", DefaultValue: "", Name: "ldap_search_scope", Type: IntType, Editable: true},
		{Scope: UserScope, Group: LdapBasicGroup, EnvironmentKey: "", DefaultValue: "", Name: "ldap_search_password", Type: PasswordType, Editable: true},
		{Scope: UserScope, Group: LdapBasicGroup, EnvironmentKey: "", DefaultValue: "", Name: "ldap_search_dn", Type: StringType, Editable: true},
		{Scope: UserScope, Group: LdapBasicGroup, EnvironmentKey: "", DefaultValue: "", Name: "ldap_verify_cert", Type: BoolType, Editable: true},
	}
)
