package proxy

import (
	"fmt"
	"github.com/goharbor/harbor/src/common/utils/log"
	"github.com/goharbor/harbor/src/core/middlewares/interceptor"
	proxyiterceptor "github.com/goharbor/harbor/src/core/middlewares/interceptor/proxy"
	"github.com/goharbor/harbor/src/core/middlewares/util"
	"net/http"
)

var (
	defaultBuilders = []interceptor.Builder{
		&manifestGetBuilder{},
	}
)

type manifestGetBuilder struct{}

func (*manifestGetBuilder) Build(req *http.Request) (interceptor.Interceptor, error) {
	log.Error("start to build intercerptor!")
	if match, _, _ := util.MatchPullManifest(req); !match {
		log.Error("failed to match Pull manifest!")
		return nil, nil
	}
	info := &util.ManifestInfo{}
	log.Info("matched pull manifest!")
	info, err := util.ParseManifestInfoFromPath(req)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifest, error %v", err)
	}
	return proxyiterceptor.NewGetManifestInterceptor(info), nil
}
