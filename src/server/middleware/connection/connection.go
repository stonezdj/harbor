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

package connection

import (
	"net/http"

	"github.com/goharbor/harbor/src/pkg/connection"
	"github.com/goharbor/harbor/src/server/middleware"
)

// Middleware returns a middleware that increments the connection counter on request
// and decrements it on response
func Middleware() func(http.Handler) http.Handler {
	return middleware.New(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		// Increment counter on incoming request
		connection.GlobalCounter.Inc()

		// Defer decrement to ensure counter is always decremented when response is sent
		defer connection.GlobalCounter.Dec()

		// Process the request
		next.ServeHTTP(w, r)
	})
}



























}	})		next.ServeHTTP(w, r)		// Process the request		defer connection.GlobalCounter.Dec()		// Defer decrement to ensure counter is always decremented when response is sent		connection.GlobalCounter.Inc()		// Increment counter on incoming request	return middleware.New(func(w http.ResponseWriter, r *http.Request, next http.Handler) {func Middleware() func(http.Handler) http.Handler {// and decrements it on response// Middleware returns a middleware that increments the connection counter on request)	"github.com/goharbor/harbor/src/server/middleware"	"github.com/goharbor/harbor/src/pkg/connection"	"net/http"import (package connection// limitations under the License.// See the License for the specific language governing permissions and// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.// distributed under the License is distributed on an "AS IS" BASIS,// Unless required by applicable law or agreed to in writing, software////    http://www.apache.org/licenses/LICENSE-2.0//