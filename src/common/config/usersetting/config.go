package usersetting

import (
	"fmt"

	"github.com/goharbor/harbor/src/adminserver/systemcfg/store/database"
)

// Manager ...
type Manager struct {
}

// Init ...
func (usm *Manager) Init() error {
	// return systemcfg.Init()
	return nil
}

// Load ...
func (usm *Manager) Load() (map[string]interface{}, error) {
	cfgStore, err := database.NewCfgStore()
	if err != nil {
		fmt.Errorf("Error occurred when Create store: %v", err)
	}
	resultMap, err := cfgStore.Read()
	fmt.Printf("Dump resultMap %+v", resultMap)
	return resultMap, err
}

// Get ....
func (usm *Manager) Get() (map[string]interface{}, error) {
	return usm.Load()
}

// Upload ...
func (usm *Manager) Upload(cfgs map[string]interface{}) error {
	cfgStore, err := database.NewCfgStore()
	if err != nil {
		fmt.Errorf("Error occurred when Create store: %v", err)
	}
	return cfgStore.Write(cfgs)
}

// Reset ...
func (usm *Manager) Reset() error {
	return nil
}
