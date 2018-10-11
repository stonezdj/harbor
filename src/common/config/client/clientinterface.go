package client

import "github.com/goharbor/harbor/src/common/models"

// ConfigClient used to retrieve configuration
type ConfigClient interface {
	GetSettingByGroup(groupName string) []models.ConfigEntry
	UpdateConfig(cfg map[string]string) error
	UpdateConfigItem(key string, value string) error
	GetConfigString(key string) (string, error)
	GetConfigInt(key string) (int, error)
	GetConfigBool(key string) (bool, error)
	GetConfigStringToStringMap(key string) (map[string]string, error)
	GetConfigMap(key string) (map[string]interface{}, error)
}
