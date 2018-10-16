package config

// ValidateFunc - function to validate configure items
type ValidateFunc func(string) (bool, error)

// Item - Configure item include default value, type, env name
type Item struct {
	//The Scope of this configuration item: eg: system, user
	Scope string
	//email, ldapbasic, ldapgroup, uaa settings, used to retieve configure items by group, for example GetLDAPBasicSetting, GetLDAPGroupSetting settings
	Group string
	//environment key to retrieves this value when initialize, for example: POSTGRESQL_HOST, only used for system settings, for user settings no EnvironmentKey
	EnvironmentKey string
	//The default string value for this key
	DefaultValue string
	//The key for current configure settings in database and rerest api
	Name string
	//It can be integer, string, bool, password, map
	Type string
	//The validation function for this field.
	Validator ValidateFunc
	//Is this settign can be modified after configure
	Editable bool
	//Reloadable - reload config from env after restart
	Reloadable bool
}

// ConfigureInterface - interface used to configure
type ConfigureInterface interface {
	// Call Validator to validate configure items, if passed, return true, else return false and return error
	Validate() (bool, error)
	// Set this configure item to configure store
	Set(value string) error
}

// ConfigureSettings - to manage all configurations
type ConfigureSettings struct {
	// ConfigureMetadata to store all metadata of configure items
	ConfigureMetaData map[string]Item
	// ConfigureValues to store all configure values
	ConfigureValues map[string]Value
}

// ConfigureValue - Configure values
type ConfigureValue struct {
	Key   string
	Value string
}

// Value -- interface to operate configure value
type Value interface {
	GetConfigString(key string) (string, error)
	GetConfigInt(key string) (int, error)
	GetConfigBool(key string) (bool, error)
	GetConfigStringToStringMap(key string) (map[string]string, error)
	GetConfigMap(key string) (map[string]interface{}, error)
}

// StorageInterface - Internal interface to manage configuration
type StorageInterface interface {
	Init() error
	//Load from store
	Load() error
	UpdateAll() error
	// Get - get all configuration item from the cached store
	Get() error
	// UpdateItem - Update a single item
	UpdateItem(item Item) error
	// UpdateItems - Update a batch of items
	UpdateItems(items []Item) error
	// Reset configure to default value
	Reset()
}

// Configures ...
var Configures ConfigureSettings

// InitMetaData ...
func (cfg *ConfigureSettings) InitMetaData() {
	metaDataMap := make(map[string]Item)
	for _, item := range ConfigList {
		metaDataMap[item.Name] = item
	}
	Configures.ConfigureMetaData = metaDataMap
}
