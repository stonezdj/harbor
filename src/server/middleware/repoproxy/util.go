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
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/goharbor/harbor/src/common/models"
	"github.com/goharbor/harbor/src/controller/blob"
	"github.com/goharbor/harbor/src/core/config"
	"github.com/goharbor/harbor/src/lib"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/replication/adapter/harbor/base"
	"github.com/goharbor/harbor/src/replication/adapter/native"
	"github.com/goharbor/harbor/src/replication/dao"
	"github.com/goharbor/harbor/src/replication/model"
	"github.com/goharbor/harbor/src/replication/registry"
	"github.com/opencontainers/go-digest"
	"net/http"
	"sync"
	"time"
)

var mu sync.Mutex
var inflight = make(map[string]interface{})

const maxWait = 10
const maxManifestWait = 40
const sleepIntervalSec = 20

func setHeaders(w http.ResponseWriter, size int64, mediaType string, dig string) {
	w.Header().Set("Content-Length", fmt.Sprintf("%v", size))
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Docker-Content-Digest", dig)
	w.Header().Set("Etag", dig)
}

// BlobExist check the blob exist in project
// TODO: use head to check exist
func BlobExist(ctx context.Context, dig string) (bool, error) {
	return blob.Ctl.Exist(ctx, dig)
}

// GetManifestFromTarget
func GetManifestFromTarget(ctx context.Context, repository string, tag string, proxyRegID int64) (distribution.Manifest, distribution.Descriptor, error) {
	desc := distribution.Descriptor{}
	adapter, err := createRegistryAdapter(proxyRegID)
	if err != nil {
		log.Error(err)
		return nil, desc, nil
	}
	man, dig, err := adapter.PullManifest(repository, tag)
	desc.Digest = digest.Digest(dig)
	return man, desc, err
}

// manifestFromTargetWithDigest ...
func manifestFromTargetWithDigest(ctx context.Context, repository string, dig string, proxyRegID int64) (distribution.Manifest, error) {
	adapter, err := createRegistryAdapter(proxyRegID)
	man, dig, err := adapter.PullManifest(repository, dig) //if tag is not provided, the digest also works
	return man, err
}

// blobFromTarget ...
func blobFromTarget(ctx context.Context, w io.Writer, repository string, dig string, proxyRegID int64) (distribution.Descriptor, error) {
	d := distribution.Descriptor{}
	adapter, err := createRegistryAdapter(proxyRegID)
	if err != nil {
		return d, err
	}

	desc, bReader, err := adapter.PullBlob(repository, dig)
	defer bReader.Close()
	if err != nil {
		log.Error(err)
	}
	written, err := io.CopyN(w, bReader, desc.Size)
	if err != nil {
		log.Error(err)
	}
	if written != desc.Size {
		log.Errorf("The size mismatch, actual:%d, expected: %d", written, desc.Size)
	}
	if string(desc.Digest) != dig {
		log.Errorf("origin dig:%v actual: %v", dig, string(desc.Digest))
	}
	d.Size = desc.Size
	d.MediaType = desc.MediaType
	d.Digest = digest.Digest(dig)
	return d, err
}

// putBlobToLocal ...
func putBlobToLocal(ctx context.Context, proxyRegID int64, orgRepo string, localRepo string, desc distribution.Descriptor, projID int64) error {
	log.Debugf("Put bl to local registry!, sourceRepo:%v, localRepo:%v, digest: %v", orgRepo, localRepo, desc.Digest)
	adapter, err := createLocalRegistryAdapter()
	if err != nil {
		log.Error(err)
		return err
	}
	orgAdapter, err := createRegistryAdapter(proxyRegID)
	if err != nil {
		log.Error(err)
		return err
	}

	_, bReader, err := orgAdapter.PullBlob(orgRepo, string(desc.Digest))
	defer bReader.Close()
	if err != nil {
		log.Error(err)
		return err
	}
	err = adapter.PushBlob(localRepo, string(desc.Digest), desc.Size, bReader)
	return err
}

// createLocalRegistryAdapter - current it only create a native adapter only,
// it should expand to other adapters for different repos
func createLocalRegistryAdapter() (*base.Adapter, error) {
	registryURL := config.GetCoreURL()
	reg := &model.Registry{
		URL: registryURL,
		Credential: &model.Credential{
			Type:         model.CredentialTypeSecret,
			AccessSecret: config.ProxyServiceSecret,
		},
	}
	return base.New(reg)
}

func createRegistryAdapter(proxyRegID int64) (*native.Adapter, error) {
	reg, err := dao.GetRegistry(proxyRegID)
	if err != nil {
		log.Error(err)
	}
	r, err := registry.FromDaoModel(reg)
	if err != nil {
		log.Error(err)
	}
	log.Infof("The credential from registry is %v", r.Credential)
	return native.NewAdapter(r), nil
}

func releaseLock(artifact string) {
	mu.Lock()
	delete(inflight, artifact)
	mu.Unlock()
}

func putManifestToLocalRepo(ctx context.Context, repo string, mfst distribution.Manifest, tag string, projectID int64) error {
	// Make sure there is only one go routing to push current artifact to local repo
	artifact := repo + ":" + tag
	mu.Lock()
	_, ok := inflight[artifact]
	if ok {
		mu.Unlock()
		// Skip to copy artifact if there is existing job running
		return nil
	}
	inflight[artifact] = 1
	mu.Unlock()
	defer releaseLock(artifact)

	adapter, err := createLocalRegistryAdapter()
	if err != nil {
		log.Error(err)
		return err
	}

	mediaType, payload, err := mfst.Payload()
	if err != nil {
		log.Error(err)
		return err
	}
	log.Infof("Pushing manifest to repo: %v, tag:%v, payload: %v", repo, tag, string(payload))
	if tag == "" {
		tag = "latest"
	}
	_, err = adapter.PushManifest(repo, tag, mediaType, payload)
	return err
}

// checkDependencies -- check all blobs used by this manifest are ready
func checkDependencies(ctx context.Context, man distribution.Manifest, dig string, mediaType string) []distribution.Descriptor {
	descriptors := man.References()
	waitDesc := make([]distribution.Descriptor, 0)
	for _, desc := range descriptors {
		log.Infof("checking the blob depedency: %v", desc.Digest)
		exist, err := BlobExist(ctx, string(desc.Digest))
		if err != nil || !exist {
			log.Info("Check dependency failed!")
			waitDesc = append(waitDesc, desc)
		}
	}

	log.Infof("Check dependency result %v", waitDesc)
	return waitDesc

}

func TrimProxyPrefix(projectName, repo string) string {
	if strings.HasPrefix(repo, projectName+"/") {
		return strings.TrimPrefix(repo, projectName+"/")
	}
	return repo
}

// updateManifestList -- Trim the manifest list, make sure all depend manifests are ready
func updateManifestList(ctx context.Context, manifest distribution.Manifest) (distribution.Manifest, error) {
	switch v := manifest.(type) {
	case *manifestlist.DeserializedManifestList:
		trimedList := make([]manifestlist.ManifestDescriptor, 0)
		for _, m := range v.Manifests {
			exist, err := BlobExist(ctx, string(m.Digest))
			if err != nil {
				continue
			}
			if exist {
				trimedList = append(trimedList, m)
			}

		}
		if len(trimedList) > 0 {
			// Avoid empty manifest in the manifest list
			return manifestlist.FromDescriptors(trimedList)
		}
	}
	return manifest, nil
}

// parseRepo parse the repo name from request url
func parseRepo(url string) string {
	u := strings.TrimPrefix(url, "/v2/")
	i := strings.LastIndex(u, "/blobs/")
	if i <= 0 {
		return url
	}
	return u[0:i]
}

// parseDigest parse the digest
func parseDigest(url string) string {
	if strings.Index(url, "sha256:") < 0 {
		return ""
	}
	parts := strings.Split(url, ":")
	if len(parts) == 2 {
		return "sha256:" + parts[1]
	}
	return ""
}

func waitAndPushManifest(contType string, ctx context.Context, man distribution.Manifest, art lib.ArtifactInfo, proj *models.Project, repo, tag string) {
	var waitBlobs []distribution.Descriptor
	n := 0
	wait := maxWait
	if contType == manifestlist.MediaTypeManifestList {
		wait = maxManifestWait
		// Make sure all depend manifests are pushed to local repo
		time.Sleep(maxManifestWait * sleepIntervalSec * time.Second)
		newMan, err := updateManifestList(ctx, man)
		if err != nil {
			log.Error(err)
		}
		err = putManifestToLocalRepo(ctx, art.ProjectName+"/"+repo, newMan, tag, proj.ProjectID)
		if err != nil {
			log.Errorf("error %v", err)
		}
		return
	}

	for n < wait {
		time.Sleep(sleepIntervalSec * time.Second)
		waitBlobs = checkDependencies(ctx, man, string(art.Digest), contType)
		if len(waitBlobs) == 0 {
			break
		}
		n = n + 1
		log.Infof("Current n=%v", n)
		if n+1 == maxWait && len(waitBlobs) > 0 && contType != manifestlist.MediaTypeManifestList {
			log.Info("Waiting blobs not empty, push it to local repo manually")
			for _, desc := range waitBlobs {
				putBlobToLocal(ctx, proj.RegistryID, repo, art.ProjectName+"/"+repo, desc, proj.ProjectID)
			}
			time.Sleep(10 * time.Second)
		}

	}
	for _, r := range man.References() {
		log.Infof("current %v, reference digest %v", art.Digest, r.Digest)
	}
	err := putManifestToLocalRepo(ctx, art.ProjectName+"/"+repo, man, tag, proj.ProjectID)
	if err != nil {
		log.Errorf("error %v", err)
	}
}

// cleanupTagInLocal cleanup delete tag from local cache
func cleanupTagInLocal(ctx context.Context, s string, s2 string) {
	log.Infof("Remove tag from repo if it is exist")
	// TODO: remove cached tag if it exist
}
