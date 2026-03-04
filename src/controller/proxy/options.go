// Copyright Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxy

import (
	proModels "github.com/goharbor/harbor/src/pkg/project/models"
	"github.com/goharbor/harbor/src/pkg/registry/interceptor/customheader"
)

type Option func(*Options)

type Options struct {
	// Speed is the data transfer speed for proxy cache from Harbor to upstream registry, no limit by default.
	Speed int32
	// CustomHeaders are HTTP headers to add to requests sent to the upstream registry (parsed from project metadata custom_request_header).
	CustomHeaders map[string]string
}

func NewOptions(opts ...Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	return o
}

func WithSpeed(speed int32) Option {
	return func(o *Options) {
		o.Speed = speed
	}
}

// WithCustomHeaders sets optional HTTP headers (key:value map) to add to upstream requests.
func WithCustomHeaders(headers map[string]string) Option {
	return func(o *Options) {
		o.CustomHeaders = headers
	}
}

// OptionsFromProject returns proxy options (speed and custom headers) from project metadata.
func OptionsFromProject(p *proModels.Project) []Option {
	if p == nil {
		return nil
	}
	opts := []Option{WithSpeed(p.ProxyCacheSpeed())}
	if v, ok := p.GetMetadata(proModels.ProMetaCustomRequestHeader); ok && v != "" {
		opts = append(opts, WithCustomHeaders(customheader.ParseCustomRequestHeader(v)))
	}
	return opts
}
