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
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/goharbor/harbor/src/common/utils/log"
	"github.com/goharbor/harbor/src/core/middlewares/interceptor"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type proxyHandler struct {
	builders []interceptor.Builder
	next     http.Handler
}

// New ...
func New(next http.Handler, builders ...interceptor.Builder) http.Handler {
	if len(builders) == 0 {
		builders = defaultBuilders
	}

	return &proxyHandler{
		builders: builders,
		next:     next,
	}
}

// MyResponseWriter ...
type MyResponseWriter struct {
	http.ResponseWriter
	Buf *bytes.Buffer
}

func (mrw *MyResponseWriter) Write(p []byte) (int, error) {
	mrw.Buf.Write(p)
	return mrw.ResponseWriter.Write(p)
}

// ServeHTTP ...
func (rh *proxyHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	log.Errorf("Handling http request %v, url: %v", req.Method, req.URL.String())

	if strings.HasSuffix(req.URL.String(), "/v2/") || req.Method != http.MethodGet {
		rh.next.ServeHTTP(rw, req)
		return
	}

	hostname := "10.160.210.111"
	username := "admin"
	password := "Harbor12345"
	repoName := "library/envoy"
	opType := "push,pull"

	//handle all get method
	token := RetrieveBearerToken(hostname, username, password, repoName, opType)

	proxyBaseUrl := &url.URL{
		Host:   hostname,
		Scheme: "https",
		Path:   "/",
	}

	log.Info("going to reverse proxy")
	proxy := httputil.NewSingleHostReverseProxy(proxyBaseUrl)

	req.URL.Scheme = "https"
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))

	bt := fmt.Sprintf("Bearer %s", token.Token)

	req.Header.Set("Authorization", bt)
	req.URL.Host = hostname

	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	mrw := &MyResponseWriter{
		ResponseWriter: rw,
		Buf:            &bytes.Buffer{},
	}

	proxy.ServeHTTP(mrw, req)

	log.Infof("Content is :" + mrw.Buf.String())

	log.Info("proxy done!")
	return

	//interceptor, err := rh.getInterceptor(req)
	//if err != nil {
	//	log.Warningf("Error occurred when to handle request in immutable handler: %v", err)
	//	http.Error(rw, util.MarshalError("InternalError", fmt.Sprintf("Error occurred when to handle request in immutable handler: %v", err)),
	//		http.StatusInternalServerError)
	//	return
	//}
	//
	//if interceptor == nil {
	//	rh.next.ServeHTTP(rw, req)
	//	return
	//}
	//
	//if err := interceptor.HandleRequest(req); err != nil {
	//	log.Warningf("Error occurred when to handle request in immutable handler: %v", err)
	//	if _, ok := err.(middlerware_err.ErrImmutable); ok {
	//		http.Error(rw, util.MarshalError("DENIED",
	//			fmt.Sprintf("%v", err)), http.StatusPreconditionFailed)
	//		return
	//	}
	//	http.Error(rw, util.MarshalError("InternalError", fmt.Sprintf("Error occurred when to handle request in immutable handler: %v", err)),
	//		http.StatusInternalServerError)
	//	return
	//}

	//mrw := MyResponseWriter{
	//	ResponseWriter: rw,
	//	Buf:            &bytes.Buffer{},
	//}
	//
	//rh.next.ServeHTTP(&mrw, req)
	//
	//log.Infof("Content is :" + mrw.Buf.String())
	//
	//digestList := ExtractDigest(mrw.Buf.Bytes())
	//for _, d := range digestList {
	//	log.Info("Digest:%v", d)
	//}

	//interceptor.HandleResponse(&mrw, req)

}

func (rh *proxyHandler) getInterceptor(req *http.Request) (interceptor.Interceptor, error) {
	for _, builder := range rh.builders {
		interceptor, err := builder.Build(req)
		if err != nil {
			return nil, err
		}

		if interceptor != nil {
			return interceptor, nil
		}
	}

	return nil, nil
}
