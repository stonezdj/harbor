package config

import (
	"github.com/goharbor/harbor/src/common/utils/log"
)

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
	InitFromString(testingMetaDataArray []Item) error
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

// InitFromArray ... Used for testing
func (s *ConfigureStore) InitFromArray(testingMetaDataArray []Item) error {
	InitMetaDataFromArray(testingMetaDataArray)
	s.ConfigureValues = map[string]Value{}
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
	resultValues := []Value{}
	for key, value := range s.ConfigureValues {
		if item, ok := ConfigureMetaData[key]; ok {
			if item.Group == groupName {
				resultValues = append(resultValues, value)
			}
		}
	}
	return resultValues, nil
}

// GetSettingByScope ...
func (s *ConfigureStore) GetSettingByScope(scope string) ([]Value, error) {
	resultValues := []Value{}
	for key, value := range s.ConfigureValues {
		if item, ok := ConfigureMetaData[key]; ok {
			if item.Scope == scope {
				resultValues = append(resultValues, value)
			}
		}
	}
	return resultValues, nil
}

// GetSetting ...
func (s *ConfigureStore) GetSetting(keyName string) (Value, error) {
	if _, ok := ConfigureMetaData[keyName]; ok {
		if value, exist := s.ConfigureValues[keyName]; exist {
			return value, nil
		} else {
			return nil, ErrValueNotSet
		}
	}
	return nil, ErrNotDefined
}

// UpdateConfig ...
func (s *ConfigureStore) UpdateConfig(cfg map[string]string) error {
	for key, value := range cfg {
		err := s.UpdateConfigValue(key, value)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateConfigValue ...
func (s *ConfigureStore) UpdateConfigValue(keyName string, value string) error {
	if _, ok := ConfigureMetaData[keyName]; ok {
		s.ConfigureValues[keyName] = &ConfigureValue{Key: keyName, Value: value}
		return nil
	} else {
		return ErrNotDefined
	}
}
