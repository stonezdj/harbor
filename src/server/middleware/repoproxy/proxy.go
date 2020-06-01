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

	"fmt"
	"strings"
	"time"

	"github.com/goharbor/harbor/src/common"
	"github.com/goharbor/harbor/src/controller/artifact"
	"github.com/goharbor/harbor/src/controller/blob"
	"github.com/goharbor/harbor/src/controller/project"
	"github.com/goharbor/harbor/src/controller/quota"
	"github.com/goharbor/harbor/src/lib"
	"github.com/goharbor/harbor/src/lib/errors"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/pkg/distribution"
	"github.com/goharbor/harbor/src/pkg/types"
	"github.com/goharbor/harbor/src/server/middleware"
	"github.com/opencontainers/go-digest"
)

func BlobGetMiddleware() func(http.Handler) http.Handler {
	return middleware.New(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		log.Infof("Request url is %v", r.URL)
		if middleware.V2BlobURLRe.MatchString(r.URL.String()) && r.Method == http.MethodGet {
			log.Infof("Getting blob with url: %v\n", r.URL.String())
			ctx := r.Context()
			projectName := parseProject(r.URL.String())
			dig := parseBlob(r.URL.String())
			repo := parseRepo(r.URL.String())
			repo = TrimProxyPrefix(repo)
			proj, err := project.Ctl.GetByName(ctx, projectName, project.Metadata(false))
			proxyRegID := proj.RegistryID
			if proxyRegID == 0 {
				next.ServeHTTP(w, r)
				return
			}
			//proxyRegID:= int64(1)
			projIDstr := fmt.Sprintf("%v", proj.ProjectID)
			if err != nil {
				log.Error(err)
			}
			log.Infof("The project id is %v", proj.ProjectID)
			log.Info(dig)
			exist, err := blob.Ctl.Exist(ctx, dig, blob.IsAssociatedWithProject(proj.ProjectID))
			if err == nil && exist {
				log.Info("The blob exist!")
			}

			if !exist {
				log.Infof("The blob doesn't exist, proxy the request to the target server, url:%v", repo)
				b, desc, err := GetBlobFromTarget(ctx, repo, dig, proxyRegID)
				log.Infof("blob digest %v, blog digest from desc:%v, digest from byte:%v", dig, desc.Digest, digest.FromBytes(b))
				if err != nil {
					log.Error(err)
					return
				}
				setHeaders(w, desc.Size, desc.MediaType, string(desc.Digest))
				w.Write(b)
				go func(desc distribution.Descriptor) {
					res := types.ResourceList{types.ResourceStorage: int64(len(b))}
					err = quota.Ctl.Request(ctx, quota.ProjectReference, projIDstr, res, func() error {
						return PutBlobToLocal(ctx, common.ProxyNamespacePrefix+repo, b, desc, proj.ProjectID)
					})
					if err != nil {
						log.Errorf("Error while puting blob to local, %v", err)
					}
				}(desc)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func setHeaders(w http.ResponseWriter, size int64, mediaType string, dig string) {
	w.Header().Set("Content-Length", fmt.Sprintf("%v", size))
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Docker-Content-Digest", dig)
	w.Header().Set("Etag", dig)
}

// BlobGetMiddleware middleware which add logger to context
func ManifestGetMiddleware() func(http.Handler) http.Handler {
	return middleware.New(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		ctx := r.Context()
		art := lib.GetArtifactInfo(ctx)
		proj, err := project.Ctl.GetByName(ctx, art.ProjectName)
		if err != nil {
			log.Error(err)
		}
		//proxyRegID := proj.ProxyRegistryID
		//if proxyRegID == 0 {
		//	next.ServeHTTP(w, r)
		//	return
		//}
		proxyRegID := int64(1)

		//projIDstr := fmt.Sprintf("%v", proj.ProjectID)
		log.Infof("Getting artifact %v", art)
		_, err = artifact.Ctl.GetByReference(ctx, art.Repository, art.Tag, nil)
		if errors.IsNotFoundErr(err) {
			log.Infof("The artifact is not found! artifact: %v", art)
			log.Info("Retrieve the artifact from proxy server")
			repo := TrimProxyPrefix(art.Repository)
			log.Infof("Repository name: %v", repo)
			log.Infof("the tag is %v", string(art.Tag))
			log.Infof("the digest is %v", string(art.Digest))
			if len(string(art.Digest)) > 0 {
				man, err := GetManifestFromTargetWithDigest(ctx, repo, string(art.Digest), 1)
				if err != nil {
					log.Error(err)
					return
				}
				ct, p, err := man.Payload()
				setHeaders(w, int64(len(p)), ct, art.Digest)
				w.Write(p)

				go func() {
					n := 0
					for n < 30 {
						time.Sleep(30 * time.Second)
						if CheckDependencies(ctx, man, string(art.Digest)) {
							break
						}
						n = n + 1
					}

					err = PutManifestToLocalRepo(ctx, common.ProxyNamespacePrefix+repo, man, "", proj.ProjectID)
					if err != nil {
						log.Errorf("error %v", err)
					}
				}()

			} else if len(string(art.Tag)) > 0 {
				man, _, err := GetManifestFromTarget(ctx, repo, string(art.Tag), proxyRegID)
				if err != nil {
					log.Error(err)
					return
				}
				ct, p, err := man.Payload()
				setHeaders(w, int64(len(p)), ct, art.Digest)
				w.Write(p)

				go func() {
					n := 0
					for n < 30 {
						time.Sleep(30 * time.Second)
						if CheckDependencies(ctx, man, art.Digest) {
							break
						}
						n = n + 1
					}

					err = PutManifestToLocalRepo(ctx, common.ProxyNamespacePrefix+repo, man, art.Tag, proj.ProjectID)

					if err != nil {
						log.Errorf("error %v", err)
					}
				}()

				return
			} else {
				log.Errorf("Invalid artifact info: %v", art)
			}

			if err != nil {
				log.Errorf("Error when fetch manifest from remote %v", err)
				return
			}

		}
		next.ServeHTTP(w, r)
	})
}

func parseProject(url string) string {
	parts := strings.Split(url, ":")
	if len(parts) == 2 {
		paths := strings.Split(parts[0], "/")
		if len(paths) > 2 {
			return paths[2]
		}
	}
	return ""
}

func parseRepo(url string) string {
	u := strings.TrimPrefix(url, "/v2/")
	i := strings.LastIndex(u, "/blobs/")
	if i <= 0 {
		return url
	}
	return u[0:i]
}

func parseBlob(url string) string {

	parts := strings.Split(url, ":")
	if len(parts) == 2 {
		return "sha256:" + parts[1]
	}
	return ""
}
