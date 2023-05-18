//  Copyright Project Harbor Authors
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package profile

import (
	"fmt"
	"net/http"
	"os"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/server/middleware"
)

const basePath = "/tmp/profile"

// Middleware middleware which validates the raw query, especially for the invalid semicolon separator.
func Middleware(skippers ...middleware.Skipper) func(http.Handler) http.Handler {
	return middleware.New(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		// if the path /tmp/profile exists, then start profiling
		if _, err := os.Stat(basePath); err == nil {
			fileName := fmt.Sprintf("%s/%s_%s_%s.out", basePath, time.Now().Format(time.RFC3339), r.Method, strings.ReplaceAll(r.URL.Path, "/", "_"))
			f, err := os.Create(fileName)
			defer f.Close()
			if err != nil {
				log.Errorf("error %v", err)
				next.ServeHTTP(w, r)
				return
			}
			pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
			next.ServeHTTP(w, r)
		} else {
			next.ServeHTTP(w, r)
		}

	}, skippers...)
}
