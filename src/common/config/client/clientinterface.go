package client

import (
	"github.com/goharbor/harbor/src/common/config"
)

// ConfigClient used to retrieve configuration
type ConfigClient interface {
	GetSettingByGroup(groupName string) []config.ConfigureValue
	GetSetting(keyName string) config.ConfigureValue
	UpdateConfig(cfg map[string]string) error
	UpdateConfigItem(key string, value string) error
}
