package rest

import (
	"github.com/goharbor/harbor/src/common/config"
)

// ConfigureDriver - use http://core:8080/api/configurations to manage configuration, commonly used outside core api container
type ConfigureDriver struct {
	config.ConfigureStore
	// ConfigURL -- URL of configure server
	ConfigURL string
}

// Load ...
func (cd *ConfigureDriver) Load() error {
	cfgs := map[string]string{}
	// Get all configure entry from configure store
	cd.LoadFromMap(cfgs)
	return nil
}

// Save ...
func (cd *ConfigureDriver) Save() error {
	return nil
}
