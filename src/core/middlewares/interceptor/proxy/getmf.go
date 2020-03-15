package proxy

import (
	"github.com/goharbor/harbor/src/common/utils/log"
	"github.com/goharbor/harbor/src/core/middlewares/interceptor"
	"github.com/goharbor/harbor/src/core/middlewares/util"
	"net/http"
)

type getmfInterceptor struct {
	mf *util.ManifestInfo
}

func (mfi *getmfInterceptor) HandleRequest(*http.Request) error {
	log.Infof("GetManifestInterceptor handleRequest, info: %#v", mfi.mf)
	return nil
}

func (mfi *getmfInterceptor) HandleResponse(writer http.ResponseWriter, req *http.Request) {

	log.Error("GetManifiestInterceptor handlerResponse")
}

// NewGetManifestInterceptor ...
func NewGetManifestInterceptor(info *util.ManifestInfo) interceptor.Interceptor {
	return &getmfInterceptor{mf: info}
}
