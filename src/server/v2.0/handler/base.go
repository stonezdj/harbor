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

package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	serror "github.com/goharbor/harbor/src/server/error"
)

// BaseAPI base API handler
type BaseAPI struct{}

// Prepare default prepare for operation
func (*BaseAPI) Prepare(ctx context.Context, operation string, params interface{}) middleware.Responder {
	return nil
}

// SendError returns response for the err
func (*BaseAPI) SendError(ctx context.Context, err error) middleware.Responder {
	return &errResponder{err: err}
}

type errResponder struct {
	err error
}

func (r *errResponder) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {
	statusCode, payload := serror.APIError(r.err)

	// payload for err is json string, so convert to raw message
	v := json.RawMessage([]byte(payload))

	rw.WriteHeader(statusCode)
	if err := producer.Produce(rw, v); err != nil {
		panic(err) // let the recovery middleware deal with this
	}
}
