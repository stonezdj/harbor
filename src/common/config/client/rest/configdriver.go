package rest

import "github.com/goharbor/harbor/src/common/config"

// Driver - use http://core:8080/api/configurations to manage configuration, commonly used outside core api container
type Driver struct {
	config.ConfigureSettings
}
