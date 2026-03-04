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

package customheader

import (
	"net/http"
	"strings"

	itcp "github.com/goharbor/harbor/src/pkg/registry/interceptor"
)

// NewInterceptor creates an interceptor that adds the given headers to every outgoing request.
func NewInterceptor(headers map[string]string) itcp.Interceptor {
	return &interceptor{headers: headers}
}

type interceptor struct {
	headers map[string]string
}

func (i *interceptor) Intercept(req *http.Request) error {
	if len(i.headers) == 0 {
		return nil
	}
	for k, v := range i.headers {
		req.Header.Set(k, v)
	}
	return nil
}

// ParseCustomRequestHeader parses a comma-separated string of "key:value" pairs into a map.
// Each item is trimmed of surrounding spaces; the first colon in each item separates key from value.
func ParseCustomRequestHeader(s string) map[string]string {
	out := make(map[string]string)
	s = strings.TrimSpace(s)
	if s == "" {
		return out
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, ":")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(part[:idx])
		value := strings.TrimSpace(part[idx+1:])
		if key != "" {
			out[key] = value
		}
	}
	return out
}
