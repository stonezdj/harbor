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

package log

import (
	"io"
	"net/http"

	"github.com/goharbor/harbor/src/common/security"
	"github.com/goharbor/harbor/src/jobservice/logger"
	"github.com/goharbor/harbor/src/lib"
	"github.com/goharbor/harbor/src/lib/log"
	tracelib "github.com/goharbor/harbor/src/lib/trace"
	"github.com/goharbor/harbor/src/server/middleware"
)

// ResponseWriter ...
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader ...
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Middleware middleware which add logger to context
func Middleware() func(http.Handler) http.Handler {
	return middleware.New(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		rid := r.Header.Get("X-Request-ID")
		if rid != "" {
			logger := log.G(r.Context())
			logger.Debugf("attach request id %s to the logger for the request %s %s", rid, r.Method, r.URL.Path)

			ctx := log.WithLogger(r.Context(), logger.WithFields(log.Fields{"requestID": rid}))
			r = r.WithContext(ctx)
		}

		traceID := tracelib.ExractTraceID(r)
		if traceID != "" {
			ctx := log.WithLogger(r.Context(), log.G(r.Context()).WithFields(log.Fields{"traceID": traceID}))
			r = r.WithContext(ctx)
		}

		enableAudit := false
		urlStr := r.URL.String()
		username := "unknown"
		var requestContent string
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete {
			enableAudit = true
			lib.NopCloseRequest(r)
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusInternalServerError)
				return
			}
			requestContent = string(body)
			if secCtx, ok := security.FromContext(r.Context()); ok {
				username = secCtx.GetUsername()
			}
		}
		rw := &ResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		if enableAudit {
			logger.Infof("the request user is %v", username)
			logger.Infof("the request Method is %v", r.Method)
			logger.Infof("the request URL is %v", urlStr)
			logger.Infof("the request body is %v", requestContent)
		}
	})
}
