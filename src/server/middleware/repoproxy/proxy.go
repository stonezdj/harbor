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

package repoproxy

import (
	"net/http"

	"github.com/goharbor/harbor/src/common/utils"
	"github.com/goharbor/harbor/src/controller/project"
	"github.com/goharbor/harbor/src/controller/proxy"
	"github.com/goharbor/harbor/src/lib"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/pkg/distribution"
	"github.com/goharbor/harbor/src/server/middleware"
)

func BlobGetMiddleware() func(http.Handler) http.Handler {
	return middleware.New(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		log.Infof("Request url is %v", r.URL)
		urlStr := r.URL.String()
		if !middleware.V2BlobURLRe.MatchString(urlStr) || r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}

		log.Infof("Getting blob with url: %v\n", urlStr)
		ctx := r.Context()
		pName := distribution.ParseProjectName(urlStr)
		dig := utils.ParseDigest(urlStr)
		repo := utils.ParseRepo(urlStr)
		repo = utils.TrimProxyPrefix(pName, repo)
		p, err := project.Ctl.GetByName(ctx, pName, project.Metadata(false))
		if err != nil {
			log.Error(err)
		}
		if proxy.Ctl.UseLocalBlob(ctx, p, dig) {
			next.ServeHTTP(w, r)
			return
		}
		log.Infof("The blob doesn't exist, proxy the request to the target server, url:%v", repo)
		err = proxy.Ctl.ProxyBlob(ctx, p, repo, dig, w)
		if err != nil {
			log.Error(err)
		}
		return
	})
}

// ManifestGetMiddleware middleware handle request for get blob request
func ManifestGetMiddleware() func(http.Handler) http.Handler {
	return middleware.New(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		ctx := r.Context()
		art := lib.GetArtifactInfo(ctx)
		p, err := project.Ctl.GetByName(ctx, art.ProjectName)
		if err != nil {
			log.Error(err)
		}

		if proxy.Ctl.UseLocalManifest(ctx, p, art) {
			next.ServeHTTP(w, r)
			return
		}

		repo := utils.TrimProxyPrefix(art.ProjectName, art.Repository)
		log.Infof("the digest is %v", string(art.Digest))
		err = proxy.Ctl.ProxyManifest(ctx, p, repo, art, w)
		if err != nil {
			log.Error(err)
		}
	})
}
