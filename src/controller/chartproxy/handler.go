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

package chartproxy

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"sigs.k8s.io/yaml"

	"github.com/goharbor/harbor/src/lib/config"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/pkg/chartproxy"
	"github.com/goharbor/harbor/src/pkg/chartproxy/model"
	"github.com/goharbor/harbor/src/pkg/proxy/secret"
	"github.com/goharbor/harbor/src/pkg/registry"
	"github.com/goharbor/harbor/src/server/router"
)

// IndexFile represents the index file in a chart repository
type IndexFile struct {
	// This is used ONLY for validation against chartmuseum's index files and is discarded after validation.
	ServerInfo map[string]interface{}         `json:"serverInfo,omitempty"`
	APIVersion string                         `json:"apiVersion"`
	Generated  time.Time                      `json:"generated"`
	Entries    map[string]model.ChartVersions `json:"entries"`
	PublicKeys []string                       `json:"publicKeys,omitempty"`

	// Annotations are additional mappings uninterpreted by Helm. They are made available for
	// other applications to add information to the index file.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// NewChartRepoHandler ...
func NewChartRepoHandler() http.Handler {
	return &ChartRepoHandler{proxyManager: chartproxy.NewManager()}
}

// ChartRepoHandler ...
type ChartRepoHandler struct {
	proxyManager chartproxy.Manager
}

func (c *ChartRepoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-yaml")
	chartList, err := c.proxyManager.ListCharts(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	indexFile := &IndexFile{
		APIVersion: "v1",
		Generated:  time.Now(),
		Entries:    make(map[string]model.ChartVersions),
	}
	for _, item := range chartList {
		chartVersions := []*model.ChartVersion{item}
		indexFile.Entries[item.Name] = chartVersions
	}

	out, err := yaml.Marshal(indexFile)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(out)
	if err != nil {
		log.Errorf("error when write response: %v", err)
	}
}

func NewChartDownloadHandler() http.Handler {
	return &ChartDownloadHandler{proxyManager: chartproxy.NewManager()}
}

type ChartDownloadHandler struct {
	proxyManager chartproxy.Manager
}

func (c ChartDownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectName := router.Param(ctx, ":project_name")
	repoName := router.Param(ctx, ":repo_name")
	repoName = strings.TrimSuffix(repoName, ".tgz")
	tag := router.Param(ctx, ":tag")
	log.Infof("The project name is %v, repo name:%v, tag: %v", projectName, repoName, tag)

	// newURL := "/v2/helm_proj/harbor/blobs/sha256:f4aed82fb0387e9108b9611ebf77fc7244dbe4e0964ddd33f484726563421650"
	registryURL := config.LocalCoreURL()
	localReg := registry.NewClientWithAuthorizer(registryURL, secret.NewAuthorizer(), true)
	repositoryName := fmt.Sprintf("%s/%s", projectName, repoName)
	digest, err := c.proxyManager.ContentDigest(ctx, repositoryName, tag)
	if err != nil {
		log.Errorf("error when get content digest: %v", err)
		return
	}
	log.Infof("the digest is %v", digest)
	sz, blobReader, err := localReg.PullBlob(fmt.Sprintf("%s/%s", projectName, repoName), digest)
	if err != nil {
		log.Errorf("error when pull blob: %v", err)
	}
	if blobReader == nil {
		log.Errorf("nil blob reader")
		return
	}
	defer blobReader.Close()
	written, err := io.CopyN(w, blobReader, sz)
	if err != nil {
		log.Errorf("error when pull blob: %v", err)
	}
	if written != sz {
		log.Errorf("The size mismatch, actual:%d, expected: %d", written, sz)
	}

	h := w.Header()
	h.Set("Content-Length", fmt.Sprintf("%v", sz))
	h.Set("Content-Type", "application/x-gtar-compressed")
}
