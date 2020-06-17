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

	"github.com/goharbor/harbor/src/controller/artifact"
	"github.com/goharbor/harbor/src/controller/project"
	"github.com/goharbor/harbor/src/lib"
	"github.com/goharbor/harbor/src/lib/errors"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/pkg/distribution"
	"github.com/goharbor/harbor/src/server/middleware"
)

func BlobGetMiddleware() func(http.Handler) http.Handler {
	return middleware.New(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		log.Infof("Request url is %v", r.URL)
		if middleware.V2BlobURLRe.MatchString(r.URL.String()) && r.Method == http.MethodGet {
			log.Infof("Getting blob with url: %v\n", r.URL.String())
			ctx := r.Context()
			pName := distribution.ParseProjectName(r.URL.String())
			dig := parseBlob(r.URL.String())
			repo := parseRepo(r.URL.String())
			repo = TrimProxyPrefix(pName, repo)
			p, err := project.Ctl.GetByName(ctx, pName, project.Metadata(false))
			proxyRegID := p.RegistryID
			if proxyRegID == 0 {
				next.ServeHTTP(w, r)
				return
			}
			if err != nil {
				log.Error(err)
			}

			exist, err := BlobExist(ctx, dig)
			if !exist {
				log.Infof("The blob doesn't exist, proxy the request to the target server, url:%v", repo)
				desc, err := GetBlobFromTarget(ctx, w, repo, dig, proxyRegID)
				if err != nil {
					log.Error(err)
					return
				}
				setHeaders(w, desc.Size, desc.MediaType, string(desc.Digest))
				go func() {
					err := PutBlobToLocal(ctx, proxyRegID, repo, pName+"/"+repo, desc, p.ProjectID)
					if err != nil {
						log.Errorf("Error while puting blob to local, %v", err)
					}
				}()
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// ManifestGetMiddleware middleware handle request for get blob request
func ManifestGetMiddleware() func(http.Handler) http.Handler {
	return middleware.New(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		ctx := r.Context()
		art := lib.GetArtifactInfo(ctx)
		proj, err := project.Ctl.GetByName(ctx, art.ProjectName)
		if err != nil {
			log.Error(err)
		}
		proxyRegID := proj.RegistryID
		if proxyRegID == 0 {
			next.ServeHTTP(w, r)
			return
		}
		log.Infof("Getting artifact %v", art)
		_, err = artifact.Ctl.GetByReference(ctx, art.Repository, art.Tag, nil)
		if !errors.IsNotFoundErr(err) {
			next.ServeHTTP(w, r)
			return
		}

		repo := TrimProxyPrefix(art.ProjectName, art.Repository)
		log.Infof("the digest is %v", string(art.Digest))
		var man distribution.Manifest
		if len(string(art.Digest)) > 0 {
			// pull by digest
			log.Infof("Getting manifest by digiest %v", art.Digest)
			exist, err := BlobExist(ctx, art.Digest)
			// exist in local, serve it with local repo
			if err == nil && exist {
				next.ServeHTTP(w, r)
				return
			}
			man, err = GetManifestFromTargetWithDigest(ctx, repo, string(art.Digest), proxyRegID)
		} else if len(string(art.Tag)) > 0 { // pull by tag
			man, _, err = GetManifestFromTarget(ctx, repo, string(art.Tag), proxyRegID)
		}

		if err != nil {
			if errors.IsNotFoundErr(err) && len(art.Tag) > 0 {
				defer CleanupTagInLocal(ctx, repo, string(art.Tag))
			}
			log.Error(err)
			return
		}
		ct, p, err := man.Payload()
		setHeaders(w, int64(len(p)), ct, art.Digest)
		w.Write(p)

		// Push manifest in background
		go func() {
			WaitAndPushManifest(ct, ctx, man, art, proj, repo, string(art.Tag))
		}()

		return

	})
}
