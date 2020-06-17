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

package proxy

import (
	"context"
	"fmt"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/goharbor/harbor/src/common/models"
	"github.com/goharbor/harbor/src/controller/artifact"
	"github.com/goharbor/harbor/src/controller/blob"
	"github.com/goharbor/harbor/src/core/config"
	"github.com/goharbor/harbor/src/lib"
	"github.com/goharbor/harbor/src/lib/errors"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/replication/adapter/harbor/base"
	"github.com/goharbor/harbor/src/replication/adapter/native"
	"github.com/goharbor/harbor/src/replication/model"
	"github.com/goharbor/harbor/src/replication/registry"
	serror "github.com/goharbor/harbor/src/server/error"
	"github.com/opencontainers/go-digest"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	maxWait          = 10
	maxManifestWait  = 40
	sleepIntervalSec = 20
)

var (
	// Ctl is a global repository controller instance
	Ctl = NewController()
)

type Controller interface {
	// UseLocalManifest check if the manifest should proxy
	UseLocalManifest(ctx context.Context, p *models.Project, art lib.ArtifactInfo) bool
	// UseLocalBlob check if the blob should proxy
	UseLocalBlob(ctx context.Context, p *models.Project, digest string) bool
	// ProxyBlob proxy the blob request to the target server
	ProxyBlob(ctx context.Context, p *models.Project, repo string, dig string, w http.ResponseWriter) error
	// ProxyManifest proxy the manifest to the target server
	ProxyManifest(ctx context.Context, p *models.Project, repo string, art lib.ArtifactInfo, w http.ResponseWriter) error
}
type controller struct {
	blobCtl     blob.Controller
	registryMgr registry.Manager
	mu          sync.Mutex
	inflight    map[string]interface{}
	artifactCtl artifact.Controller
}

func (c *controller) UseLocalManifest(ctx context.Context, p *models.Project, art lib.ArtifactInfo) bool {
	if p.RegistryID < 1 {
		return true
	}
	reg, err := c.registryMgr.Get(p.RegistryID)
	if err != nil {
		return true
	}
	if reg.Status != model.Healthy {
		return true
	}
	if len(string(art.Digest)) > 0 {
		exist, err := c.blobExist(ctx, art.Digest)
		if err == nil && exist {
			return true
		}
	}
	return false
}

func (c *controller) UseLocalBlob(ctx context.Context, p *models.Project, digest string) bool {
	if p.RegistryID < 1 {
		return true
	}
	reg, err := c.registryMgr.Get(p.RegistryID)
	if err != nil {
		return true
	}
	if reg.Status != model.Healthy {
		return true
	}
	exist, err := c.blobExist(ctx, digest)
	if err != nil {
		return false
	}
	return exist
}

func (c *controller) ProxyManifest(ctx context.Context, p *models.Project, repo string, art lib.ArtifactInfo, w http.ResponseWriter) error {
	var man distribution.Manifest
	var err error
	if len(string(art.Digest)) > 0 {
		// pull by digest
		log.Infof("Getting manifest by digiest %v", art.Digest)
		man, err = c.manifestFromTargetWithDigest(ctx, repo, string(art.Digest), p.RegistryID)
	} else if len(string(art.Tag)) > 0 { // pull by tag
		man, _, err = c.manifestFromTarget(ctx, repo, string(art.Tag), p.RegistryID)
	}

	if err != nil {
		if errors.IsNotFoundErr(err) && len(art.Tag) > 0 {
			go func() {
				c.cleanupTagInLocal(ctx, repo, string(art.Tag))
			}()
		}
		serror.SendError(w, err)
		return err
	}

	ct, payload, err := man.Payload()
	setHeaders(w, int64(len(payload)), ct, art.Digest)
	w.Write(payload)

	// Push manifest in background
	go func() {
		c.waitAndPushManifest(ct, ctx, man, art, p, repo, string(art.Tag))
	}()

	return nil
}

func (c *controller) ProxyBlob(ctx context.Context, p *models.Project, repo string, dig string, w http.ResponseWriter) error {
	log.Infof("The blob doesn't exist, proxy the request to the target server, url:%v", repo)
	desc, err := c.blobFromTarget(ctx, w, repo, dig, p.RegistryID)
	if err != nil {
		log.Error(err)
		serror.SendError(w, err)
		return err
	}
	setHeaders(w, desc.Size, desc.MediaType, string(desc.Digest))

	go func() {
		err := c.putBlobToLocal(ctx, p.RegistryID, repo, p.Name+"/"+repo, desc, p.ProjectID)
		if err != nil {
			log.Errorf("Error while puting blob to local, %v", err)
		}
	}()
	return nil
}

func NewController() Controller {
	return &controller{
		blobCtl:     blob.Ctl,
		registryMgr: registry.NewDefaultManager(),
		inflight:    make(map[string]interface{}),
		artifactCtl: artifact.Ctl,
	}
}

func (c *controller) blobFromTarget(ctx context.Context, w io.Writer, repository string, dig string, proxyRegID int64) (distribution.Descriptor, error) {
	d := distribution.Descriptor{}
	adapter, err := c.createRegistryAdapter(proxyRegID)
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

func (c *controller) putBlobToLocal(ctx context.Context, proxyRegID int64, orgRepo string, localRepo string, desc distribution.Descriptor, projID int64) error {
	log.Debugf("Put blob to local registry!, sourceRepo:%v, localRepo:%v, digest: %v", orgRepo, localRepo, desc.Digest)
	adapter, err := c.createLocalRegistryAdapter()
	if err != nil {
		log.Error(err)
		return err
	}
	orgAdapter, err := c.createRegistryAdapter(proxyRegID)
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

func (c *controller) putManifestToLocalRepo(ctx context.Context, repo string, mfst distribution.Manifest, tag string, projectID int64) error {
	// Make sure there is only one go routing to push current artifact to local repo
	artifact := repo + ":" + tag
	c.mu.Lock()
	_, ok := c.inflight[artifact]
	if ok {
		c.mu.Unlock()
		// Skip to copy artifact if there is existing job running
		return nil
	}
	c.inflight[artifact] = 1
	c.mu.Unlock()
	defer c.releaseLock(artifact)

	adapter, err := c.createLocalRegistryAdapter()
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

func (c *controller) manifestFromTargetWithDigest(ctx context.Context, repository string, dig string, proxyRegID int64) (distribution.Manifest, error) {
	adapter, err := c.createRegistryAdapter(proxyRegID)
	man, dig, err := adapter.PullManifest(repository, dig)
	return man, err
}

func (c *controller) manifestFromTarget(ctx context.Context, repository string, tag string, proxyRegID int64) (distribution.Manifest, distribution.Descriptor, error) {
	desc := distribution.Descriptor{}
	adapter, err := c.createRegistryAdapter(proxyRegID)
	if err != nil {
		log.Error(err)
		return nil, desc, err
	}
	man, dig, err := adapter.PullManifest(repository, tag)
	desc.Digest = digest.Digest(dig)
	return man, desc, err
}

func (c *controller) blobExist(ctx context.Context, dig string) (bool, error) {
	return c.blobCtl.Exist(ctx, dig)
}

func setHeaders(w http.ResponseWriter, size int64, mediaType string, dig string) {
	w.Header().Set("Content-Length", fmt.Sprintf("%v", size))
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Docker-Content-Digest", dig)
	w.Header().Set("Etag", dig)
}

func (c *controller) createRegistryAdapter(proxyRegID int64) (*native.Adapter, error) {
	reg, err := registry.NewDefaultManager().Get(proxyRegID)
	if err != nil {
		return nil, err
	}
	log.Infof("The credential from registry is %v", reg.Credential)
	return native.NewAdapter(reg), nil
}

// checkDependencies -- check all blobs used by this manifest are ready
func (c *controller) checkDependencies(ctx context.Context, man distribution.Manifest, dig string, mediaType string) []distribution.Descriptor {
	descriptors := man.References()
	waitDesc := make([]distribution.Descriptor, 0)
	for _, desc := range descriptors {
		log.Infof("checking the blob depedency: %v", desc.Digest)
		exist, err := c.blobExist(ctx, string(desc.Digest))
		if err != nil || !exist {
			log.Info("Check dependency failed!")
			waitDesc = append(waitDesc, desc)
		}
	}

	log.Infof("Check dependency result %v", waitDesc)
	return waitDesc

}

// createLocalRegistryAdapter - current it only create a native adapter only,
// it should expand to other adapters for different repos
func (c *controller) createLocalRegistryAdapter() (*base.Adapter, error) {
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

func (c *controller) releaseLock(artifact string) {
	c.mu.Lock()
	delete(c.inflight, artifact)
	c.mu.Unlock()
}

// cleanupTagInLocal cleanup delete tag from local cache
func (c *controller) cleanupTagInLocal(ctx context.Context, s string, s2 string) {
	log.Infof("Remove tag from repo if it is exist")
	// TODO: remove cached tag if it exist
}

func (c *controller) waitAndPushManifest(contType string, ctx context.Context, man distribution.Manifest, art lib.ArtifactInfo, proj *models.Project, repo, tag string) {
	var waitBlobs []distribution.Descriptor
	n := 0
	wait := maxWait
	if contType == manifestlist.MediaTypeManifestList {
		wait = maxManifestWait
		// Make sure all depend manifests are pushed to local repo
		time.Sleep(maxManifestWait * sleepIntervalSec * time.Second)
		newMan, err := c.updateManifestList(ctx, man)
		if err != nil {
			log.Error(err)
		}
		err = c.putManifestToLocalRepo(ctx, art.ProjectName+"/"+repo, newMan, tag, proj.ProjectID)
		if err != nil {
			log.Errorf("error %v", err)
		}
		return
	}

	for n < wait {
		time.Sleep(sleepIntervalSec * time.Second)
		waitBlobs = c.checkDependencies(ctx, man, string(art.Digest), contType)
		if len(waitBlobs) == 0 {
			break
		}
		n = n + 1
		log.Infof("Current n=%v", n)
		if n+1 == maxWait && len(waitBlobs) > 0 && contType != manifestlist.MediaTypeManifestList {
			log.Info("Waiting blobs not empty, push it to local repo manually")
			for _, desc := range waitBlobs {
				c.putBlobToLocal(ctx, proj.RegistryID, repo, art.ProjectName+"/"+repo, desc, proj.ProjectID)
			}
			time.Sleep(10 * time.Second)
		}

	}
	for _, r := range man.References() {
		log.Infof("current %v, reference digest %v", art.Digest, r.Digest)
	}
	err := c.putManifestToLocalRepo(ctx, art.ProjectName+"/"+repo, man, tag, proj.ProjectID)
	if err != nil {
		log.Errorf("error %v", err)
	}
}

// updateManifestList -- Trim the manifest list, make sure all depend manifests are ready
func (c *controller) updateManifestList(ctx context.Context, manifest distribution.Manifest) (distribution.Manifest, error) {
	switch v := manifest.(type) {
	case *manifestlist.DeserializedManifestList:
		trimedList := make([]manifestlist.ManifestDescriptor, 0)
		for _, m := range v.Manifests {
			exist, err := c.blobExist(ctx, string(m.Digest))
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
