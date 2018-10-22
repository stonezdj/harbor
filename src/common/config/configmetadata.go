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

	//Type
	IntType      = "int"
	Int64Type    = "int64"
	StringType   = "string"
	BoolType     = "bool"
	PasswordType = "password"
	MapType      = "map"
)

// ValidateFunc - function to validate configure items
type ValidateFunc func(key, value string) error

// Item - Configure item include default value, type, env name
type Item struct {
	//The Scope of this configuration item: eg: system, user
	Scope string `json:"scope,omitempty"`
	//email, ldapbasic, ldapgroup, uaa settings, used to retieve configure items by group, for example GetLDAPBasicSetting, GetLDAPGroupSetting settings
	Group string `json:"group,omitempty"`
	//environment key to retrieves this value when initialize, for example: POSTGRESQL_HOST, only used for system settings, for user settings no EnvironmentKey
	EnvironmentKey string `json:"environment_key,omitempty"`
	//The default string value for this key
	DefaultValue string `json:"default_value,omitempty"`
	//Has default value
	HasDefaultValue bool `json:"has_default_value,omitempty"`
	//The key for current configure settings in database and rerest api
	Name string `json:"name,omitempty"`
	//It can be integer, string, bool, password, map
	Type string `json:"type,omitempty"`
	//The validation function for this field.
	Validator ValidateFunc `json:"validator,omitempty"`
	//Is this settign can be modified after configure
	Editable bool `json:"editable,omitempty"`
	//Reloadable - reload config from env after restart
	Reloadable bool `json:"reloadable,omitempty"`
}

// ConfigureMetaData ...
var ConfigureMetaData map[string]Item

// InitMetaData ...
func InitMetaData() {
	metaDataMap := make(map[string]Item)
	for _, item := range ConfigList {
		metaDataMap[item.Name] = item
	}
	ConfigureMetaData = metaDataMap
}

// InitMetaDataFromArray - used for testing
func InitMetaDataFromArray(items []Item) {
	resultMap := map[string]Item{}
	for _, item := range items {
		resultMap[item.Name] = item
	}
	ConfigureMetaData = resultMap
}
