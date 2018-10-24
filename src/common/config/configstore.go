package config

import (
	"os"
	"sync"

	"github.com/goharbor/harbor/src/common/utils/log"
)

// ConfigureStore - to manage all configurations
type ConfigureStore struct {
	sync.RWMutex
	// ConfigureValues to store all configure values
	configureValues map[string]Value
}

// NewConfigureStore ...
func NewConfigureStore() *ConfigureStore {
	cs := new(ConfigureStore)
	cs.configureValues = make(map[string]Value)
	return cs
}

func (s *ConfigureStore) readMap(key string) (Value, error) {
	s.RLock()
	defer s.RUnlock()
	if value, ok := s.configureValues[key]; ok {
		return value, nil
	}
	return nil, ErrValueNotSet

}

func (s *ConfigureStore) writeMap(key string, value Value) {
	s.Lock()
	defer s.Unlock()
	s.configureValues[key] = value
}

// StorageInterface ...
type StorageInterface interface {
	// Init - init configurations with default value
	Init() error
	// InitFromString - used for testing
	InitFromString(testingMetaDataArray []Item) error
	// Load from store
	Load() error
	// Save to store
	Save() error
	// LoadFromMap ...
	LoadFromMap(map[string]string)
	// Save all configuration to store
	UpdateAll() error
	// Reset configure to default value
	Reset()
}

// Init - int the store
func (s *ConfigureStore) Init() error {
	MetaData.InitMetaData()
	// Init Default Value
	itemArray := MetaData.GetAllConfigureItems()
	for _, item := range itemArray {
		if item.HasDefaultValue {
			c := &ConfigureValue{item.Name, item.DefaultValue}
			err := c.Validate()
			if err == nil {
				s.writeMap(item.Name, c)
			} else {
				log.Errorf("Failed to init config item %+v, default err: %+v", c, err)
			}
		}
	}

	// Init System Value
	for _, item := range itemArray {
		if item.Scope == SystemScope {
			if len(item.EnvironmentKey) > 0 {
				if envValue, ok := os.LookupEnv(item.EnvironmentKey); ok {
					c := &ConfigureValue{item.Name, envValue}
					err := c.Validate()
					if err == nil {
						s.writeMap(item.Name, c)
					} else {
						log.Errorf("Failed to init system config item %+v,  err: %+v", c, err)
					}
				}
			}
		}
	}

	return nil
}

// LoadFromMap ...
func (s *ConfigureStore) LoadFromMap(cfgs map[string]string) {
	for k, v := range cfgs {
		c := &ConfigureValue{k, v}
		err := c.Validate()
		if err == nil {
			s.writeMap(k, c)
		} else {
			log.Errorf("Failed LoadFromMap, config item %+v,  err: %+v", c, err)
		}
	}
}

// InitFromArray ... Used for testing
func (s *ConfigureStore) InitFromArray(testingMetaDataArray []Item) error {
	MetaData.InitMetaDataFromArray(testingMetaDataArray)
	itemArray := MetaData.GetAllConfigureItems()
	// Init Default Value
	for _, item := range itemArray {
		if item.HasDefaultValue {
			c := &ConfigureValue{item.Name, item.DefaultValue}
			err := c.Validate()
			if err == nil {
				s.writeMap(item.Name, c)
			} else {
				log.Errorf("Failed InitFromArray, config item %+v,  err: %+v", c, err)
			}
		}
	}

	// Init System Value
	for _, item := range itemArray {
		if item.Scope == SystemScope {
			if len(item.EnvironmentKey) > 0 {
				if envValue, ok := os.LookupEnv(item.EnvironmentKey); ok {
					c := &ConfigureValue{item.Name, envValue}
					err := c.Validate()
					if err == nil {
						s.writeMap(item.Name, c)
					} else {
						log.Errorf("Failed InitFromArray, config item %+v,  err: %+v", c, err)
					}
				}
			}
		}
	}

	return nil
}

// Load ...
func (s *ConfigureStore) Load() error {
	panic("not implemented")
}

// Save ...
func (s *ConfigureStore) Save() error {
	panic("not implemented")
}

// UpdateAll ...
func (s *ConfigureStore) UpdateAll() error {
	panic("not implemented")
}

// Reset ...
func (s *ConfigureStore) Reset() {
	s.Lock()
	defer s.Unlock()
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

// GetAllSettings ...
func (s *ConfigureStore) GetAllSettings() ([]Value, error) {
	s.RLock()
	defer s.RUnlock()
	resultValues := []Value{}
	for _, value := range s.configureValues {
		resultValues = append(resultValues, value)
	}
	return resultValues, nil
}

// GetSettingByGroup ...
func (s *ConfigureStore) GetSettingByGroup(groupName string) ([]Value, error) {
	resultValues := []Value{}
	s.RLock()
	defer s.RUnlock()

	for key, value := range s.configureValues {
		item, err := MetaData.GetConfigMetaData(key)
		if err == nil {
			if item.Group == groupName {
				resultValues = append(resultValues, value)
			}
		} else {
			return nil, err
		}
	}

	return resultValues, nil
}

// GetSettingByScope ...
func (s *ConfigureStore) GetSettingByScope(scope string) ([]Value, error) {
	s.RLock()
	defer s.RUnlock()
	resultValues := []Value{}
	for key, value := range s.configureValues {
		item, err := MetaData.GetConfigMetaData(key)
		if err == nil {
			if item.Scope == scope {
				resultValues = append(resultValues, value)
			}
		} else {
			return nil, err
		}
	}
	return resultValues, nil
}

// GetSetting ...
func (s *ConfigureStore) GetSetting(keyName string) (Value, error) {
	s.RLock()
	defer s.RUnlock()
	_, err := MetaData.GetConfigMetaData(keyName)
	if err == nil {
		return s.readMap(keyName)
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

	_, err := MetaData.GetConfigMetaData(keyName)
	if err == nil {
		c := &ConfigureValue{Key: keyName, Value: value}
		err := c.Validate()
		if err != nil {
			return err
		}
		s.writeMap(keyName, c)
	}
	return err
}
