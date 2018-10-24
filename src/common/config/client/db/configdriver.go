package db

import (
	"github.com/goharbor/harbor/src/common/config"
	"github.com/goharbor/harbor/src/common/dao"
	"github.com/goharbor/harbor/src/common/models"
)

// ConfigureDriver - Retrieve configurations from database
type ConfigureDriver struct {
	config.ConfigureStore
}

// NewDBConfigureStore ...
func NewDBConfigureStore() *ConfigureDriver {
	return NewDBConfigureStoreFromArray(config.ConfigList)
}

// NewDBConfigureStoreFromArray ...
func NewDBConfigureStoreFromArray(items []config.Item) *ConfigureDriver {
	cd := &ConfigureDriver{*config.NewConfigureStore()}
	config.MetaData.InitMetaDataFromArray(items)
	cd.InitFromArray(items)
	return cd
}

// Load ...
func (cd *ConfigureDriver) Load() error {
	cfgs := map[string]string{}
	configEntries, err := dao.GetConfigEntries()
	if err != nil {
		return err
	}
	for _, item := range configEntries {
		cfgs[item.Key] = item.Value
	}
	cd.LoadFromMap(cfgs)
	return nil
}

// Save ...
func (cd *ConfigureDriver) Save() error {
	var configEntries []models.ConfigEntry
	configValues, err := cd.GetAllSettings()
	if err != nil {
		return err
	}
	for _, v := range configValues {
		var entry = new(models.ConfigEntry)
		entry.Key = v.GetKey()
		entry.Value = v.GetString()
		configEntries = append(configEntries, *entry)
	}
	return dao.SaveConfigEntries(configEntries)
}
