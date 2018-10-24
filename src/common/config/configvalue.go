package config

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/goharbor/harbor/src/common/utils/log"
)

var (
	// ErrNotDefined ...
	ErrNotDefined = errors.New("configure item is not defined in metadata")
	// ErrTypeNotMatch ...
	ErrTypeNotMatch = errors.New("the required value doesn't matched with metadata defined")
	// ErrInvalidData ...
	ErrInvalidData = errors.New("the data provided is invalid")
	// ErrValueNotSet ...
	ErrValueNotSet = errors.New("the configure value is not set")
)

// ConfigureValue - Configure values
type ConfigureValue struct {
	Key   string
	Value string
}

// Value -- interface to operate configure value
type Value interface {
	GetString() string
	// GetInt - return the int value of current value
	GetInt() int
	// GetInt64 - return the int64 value of current value
	GetInt64() int64
	// GetBool - return the bool value of current setting
	GetBool() bool
	// GetStringToStringMap - return the string to string map of current value
	GetStringToStringMap() map[string]string
	// GetMap - return the map of current value
	GetMap() map[string]interface{}
	// Validator to validate configure items, if passed, return true, else return false and return error
	Validate() error
	// Set this configure item to configure store
	Set(key, value string) error

	GetKey() string
}

// GetString - Get the string value of current configure
func (c *ConfigureValue) GetString() string {
	//Any type has the string value
	_, err := MetaData.GetConfigMetaData(c.Key)
	if err == nil {
		return c.Value
	}
	return ""
}

// GetKey ...
func (c *ConfigureValue) GetKey() string {
	return c.Key
}

// GetInt - return the int value of current value
func (c *ConfigureValue) GetInt() int {
	if item, err := MetaData.GetConfigMetaData(c.Key); err == nil {
		if item.Type == IntType {
			result, err := strconv.Atoi(c.Value)
			if err == nil {
				return result
			}
		}
	}
	log.Errorf("The current value can not convert to integer, %+v", c)
	return 0
}

// GetInt64 - return the int64 value of current value
func (c *ConfigureValue) GetInt64() int64 {
	if item, err := MetaData.GetConfigMetaData(c.Key); err == nil {
		if (item.Type == IntType) || (item.Type == Int64Type) {
			result, err := strconv.ParseInt(c.Value, 10, 64)
			if err == nil {
				return result
			}
		}
	}
	log.Errorf("The current value can not convert to integer, %+v", c)
	return 0
}

// GetBool - return the bool value of current setting
func (c *ConfigureValue) GetBool() bool {
	if item, err := MetaData.GetConfigMetaData(c.Key); err == nil {
		if item.Type == BoolType {
			result, err := strconv.ParseBool(c.Value)
			if err == nil {
				return result
			}
		}
	}
	log.Errorf("The current value can not convert to bool, %+v", c)
	return false
}

// GetStringToStringMap - return the string to string map of current value
func (c *ConfigureValue) GetStringToStringMap() map[string]string {
	result := map[string]string{}
	if item, err := MetaData.GetConfigMetaData(c.Key); err == nil {
		if item.Type == MapType {
			err := json.Unmarshal([]byte(c.Value), &result)
			if err == nil {
				return result
			}
		}
	}
	log.Errorf("The current value can not convert to map[string]string, %+v", c)
	return result
}

// GetMap - return the map of current value
func (c *ConfigureValue) GetMap() map[string]interface{} {
	result := map[string]interface{}{}
	if item, err := MetaData.GetConfigMetaData(c.Key); err == nil {
		if item.Type == MapType {
			err := json.Unmarshal([]byte(c.Value), &result)
			if err == nil {
				return result
			}
		}
	}
	log.Errorf("The current value can not convert to map[string]interface{}, %+v", c)
	return result
}

// Validate - to validate configure items, if passed, return true, else return false and return error
func (c *ConfigureValue) Validate() error {
	if item, err := MetaData.GetConfigMetaData(c.Key); err == nil {
		if item.Validator != nil {
			return item.Validator(c.Key, c.Value)
		}
		return nil
	}
	return ErrNotDefined
}

// Set - set this configure item to configure store
func (c *ConfigureValue) Set(key, value string) error {
	if item, err := MetaData.GetConfigMetaData(c.Key); err == nil {
		if item.Validator != nil {
			err := item.Validator(key, value)
			if err != nil {
				return ErrInvalidData
			}
		}
		c.Key = key
		c.Value = value
		return nil
	}
	return ErrNotDefined
}
