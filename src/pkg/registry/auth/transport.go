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

package auth

import "net/http"

// headerTransport wraps a RoundTripper and adds custom headers to every request.
// Used so authorizer-originated requests (e.g. auth discovery, token fetch) also get proxy custom headers.
type headerTransport struct {
	base   http.RoundTripper
	headers map[string]string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	return t.base.RoundTrip(req)
}

// NewHeaderTransport returns a RoundTripper that adds the given headers to every request.
// If headers is nil or empty, base is returned unchanged.
func NewHeaderTransport(base http.RoundTripper, headers map[string]string) http.RoundTripper {
	if base == nil {
		return nil
	}
	if len(headers) == 0 {
		return base
	}
	return &headerTransport{base: base, headers: headers}
}
