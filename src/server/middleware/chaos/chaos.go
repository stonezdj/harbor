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

package chaos

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/goharbor/harbor/src/lib/errors"
	lib_http "github.com/goharbor/harbor/src/lib/http"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/server/middleware"
)

const (
	mock404        = "mock404"
	mockNoResponse = "mocknoresponse"
)

// Configure struct for chaos config
type Configure struct {
	Type    string
	Method  string
	Pattern string
	Code    int
}

func config() *Configure {
	cfg := os.Getenv("CHAOS_CFG")
	if len(cfg) == 0 {
		return nil
	}

	cfg = strings.ToLower(cfg)

	if strings.HasPrefix(cfg, mock404) {
		parts := strings.SplitN(cfg, ":", 4)
		code, err := strconv.ParseInt(parts[3], 10, 64)
		if err != nil {
			log.Errorf("failed to parse http code in %v", parts[2])
			return nil
		}
		return &Configure{
			Type:    mock404,
			Method:  strings.ToUpper(parts[1]),
			Pattern: parts[2],
			Code:    int(code),
		}
	}

	if strings.HasPrefix(cfg, mockNoResponse) {
		parts := strings.SplitN(cfg, ":", 3)
		return &Configure{
			Type:    mockNoResponse,
			Method:  strings.ToUpper(parts[1]),
			Pattern: parts[2],
		}
	}
	return nil

}

// Middleware mock chaos
func Middleware(skippers ...middleware.Skipper) func(http.Handler) http.Handler {
	return middleware.New(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		if r.URL != nil {
			log.Errorf("chaos middleware: request %v, method:%v", r.URL, r.Method)
		}
		cfg := config()
		if cfg != nil {
			if cfg.Type == mock404 {
				if r.Method == cfg.Method && strings.HasPrefix(r.URL.String(), cfg.Pattern) {
					MockHTTPCode(w, r, cfg)
					return
				}
			} else if r.Method == cfg.Method && cfg.Type == mockNoResponse {
				MockNoResponse(w, r)
			}
		}
		next.ServeHTTP(w, r)
	}, skippers...)
}

// MockHTTPCode ...
func MockHTTPCode(w http.ResponseWriter, r *http.Request, cfg *Configure) {
	if cfg.Code == http.StatusNotFound {
		lib_http.SendError(w, errors.New("not found from chaos middleware").WithCode(errors.NotFoundCode))
	} else if cfg.Code == http.StatusInternalServerError {
		lib_http.SendError(w, errors.New("internal server error from chaos middleware").WithCode(errors.GeneralCode))
	} else if cfg.Code == http.StatusTooManyRequests {
		lib_http.SendError(w, errors.New("too many requests from chaos middleware").WithCode(errors.TooManyRequestCode))
	} else if cfg.Code == http.StatusForbidden {
		lib_http.SendError(w, errors.New("forbidden from chaos middleware").WithCode(errors.ForbiddenCode))
	}

}

// MockNoResponse ...
func MockNoResponse(w http.ResponseWriter, r *http.Request) {
	time.Sleep(30 * time.Minute)
}
