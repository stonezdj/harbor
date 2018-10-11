package config

// ValidateFunc - function to validate configure items
type ValidateFunc func(string) (bool, error)

// Item - Configure item include default value, type, env name
type Item struct {
	//true for system, false for user settings
	SystemConfig bool
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
	// GetString value of this configure item
	GetString() string
	// GetName get the name of current configure item
	GetName() string
	// GetInt get the int value of current item.
	GetInt() int
	// GetPassword get the decrypted password value
	GetPassword() string
	// GetMap get the map value of current item.
	GetMap() map[string]interface{}
	// GetStringToStringMap get the string to string map of current value
	GetStringToStringMap() map[string]string
}

// ConfigureSettings - to manage all configurations
type ConfigureSettings struct {
	// ConfigureMetadata to store all metadata of configure items
	ConfigureMetaData map[string]Item
	// ConfigureValues to store all configure values
	ConfigureValues map[string]string
}

// StorageInterface - Internal interface to manage configuration
type StorageInterface interface {
	Init() error
	Load() error
	UpdateAll() error
	// Get - get all configuration item from store
	Get() error
	// UpdateItem - Update a single item
	UpdateItem(item Item) error
	// UpdateItems - Update a batch of items
	UpdateItems(items []Item) error
	// Reset configure to default value
	Reset()
}
