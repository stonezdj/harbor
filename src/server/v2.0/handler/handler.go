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
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/go-openapi/errors"
	"github.com/goharbor/harbor/src/server/v2.0/restapi"
)

// New returns http handler for API V2.0
func New() http.Handler {
	h, api, err := restapi.HandlerAPI(restapi.Config{
		ArtifactAPI: NewArtifactAPI(),
	})
	if err != nil {
		log.Fatal(err)
	}

	api.ServeError = serveError

	return h
}

type apiError struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

type ociError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Before executing operation handler, go-swagger will bind a parameters object to a request and validate the request,
// it will return directly when bind and validate failed.
// The response format of the default ServeError implementation does not match the OCI error response format.
// So we needed to convert the format to the OCI error response format.
func serveError(rw http.ResponseWriter, r *http.Request, err error) {
	w := httptest.NewRecorder()
	errors.ServeError(w, r, err)

	rw.WriteHeader(w.Code)
	for key, value := range w.Header() {
		rw.Header().Set(key, value[0])
	}

	var er apiError
	json.Unmarshal(w.Body.Bytes(), &er)

	code := strings.Replace(strings.ToUpper(http.StatusText(w.Code)), " ", "_", -1)
	errs := struct {
		Errors []ociError `json:"errors"`
	}{
		[]ociError{{Code: code, Message: er.Message}},
	}

	body, _ := json.Marshal(errs)
	rw.Write(body)
}
