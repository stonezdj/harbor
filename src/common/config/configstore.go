package config

import "github.com/goharbor/harbor/src/common/utils/log"

// ConfigureStore - to manage all configurations
type ConfigureStore struct {
	// ConfigureValues to store all configure values
	ConfigureValues map[string]Value
}

// StorageInterface ...
type StorageInterface interface {
	// Init - init configurations with default value
	Init() error
	// InitFromString - used for testing
	InitFromString(metadataJSONString string) error
	// Load from store
	Load() error
	// Save all configuration to store
	UpdateAll() error
	// Reset configure to default value
	Reset()
}

// Init - int the store
func (s *ConfigureStore) Init() error {
	InitMetaData()
	for k, v := range ConfigureMetaData {
		if v.HasDefaultValue {
			s.ConfigureValues[k] = &ConfigureValue{k, v.DefaultValue}
		}
	}
	return nil
}

// InitFromString ... Used for testing
func (s *ConfigureStore) InitFromString(metadataJSONString string) error {
	InitMetaDataFromJSONString(metadataJSONString)
	for k, v := range ConfigureMetaData {
		if v.HasDefaultValue {
			s.ConfigureValues[k] = &ConfigureValue{k, v.DefaultValue}
		}
	}
	return nil
}

// Load ...
func (s *ConfigureStore) Load() error {
	panic("not implemented")
}

// UpdateAll ...
func (s *ConfigureStore) UpdateAll() error {
	panic("not implemented")
}

// Reset ...
func (s *ConfigureStore) Reset() {
	err := s.Init()
	if err != nil {
		log.Errorf("Error occurred when Init: %v", err)
		return
	}
	err = s.UpdateAll()
	if err != nil {
		log.Errorf("Error occurred when UpdateAll: %v", err)
	}
}

// GetSettingByGroup ...
func (s *ConfigureStore) GetSettingByGroup(groupName string) ([]Value, error) {
	panic("not implemented")
}

// GetSettingByScope ...
func (s *ConfigureStore) GetSettingByScope(scope string) ([]Value, error) {
	panic("not implemented")
}

// GetSetting ...
func (s *ConfigureStore) GetSetting(keyName string) (Value, error) {
	panic("not implemented")
}

// UpdateConfig ...
func (s *ConfigureStore) UpdateConfig(cfg map[string]string) error {
	panic("not implemented")
}

// UpdateConfigValue ...
func (s *ConfigureStore) UpdateConfigValue(key string, value string) error {
	panic("not implemented")
}
