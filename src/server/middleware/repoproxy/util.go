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
	"io"
	"strings"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/goharbor/harbor/src/controller/blob"
	"github.com/goharbor/harbor/src/core/config"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/replication/adapter/harbor/base"
	"github.com/goharbor/harbor/src/replication/adapter/native"
	"github.com/goharbor/harbor/src/replication/dao"
	"github.com/goharbor/harbor/src/replication/model"
	"github.com/goharbor/harbor/src/replication/registry"
	"github.com/opencontainers/go-digest"
)

func GetManifestFromTarget(ctx context.Context, repository string, tag string, proxyRegID int64) (distribution.Manifest, distribution.Descriptor, error) {
	desc := distribution.Descriptor{}
	adapter, err := CreateRegistryAdapter(proxyRegID)
	if err != nil {
		log.Error(err)
		return nil, desc, nil
	}
	man, dig, err := adapter.PullManifest(repository, tag)
	desc.Digest = digest.Digest(dig)
	return man, desc, nil
}

func GetManifestFromTargetWithDigest(ctx context.Context, repository string, dig string, proxyRegID int64) (distribution.Manifest, error) {
	adapter, err := CreateRegistryAdapter(proxyRegID)
	man, dig, err := adapter.PullManifest(repository, dig) //if tag is not provided, the digest also works
	return man, err
}

func GetBlobFromTarget(ctx context.Context, w io.Writer, repository string, dig string, proxyRegID int64) (distribution.Descriptor, error) {
	d := distribution.Descriptor{}
	adapter, err := CreateRegistryAdapter(proxyRegID)
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

func PutBlobToLocal(ctx context.Context, proxyRegID int64, orgRepo string, localRepo string, desc distribution.Descriptor, projID int64) error {
	log.Debugf("Put bl to local registry!, sourceRepo:%v, localRepo:%v, digest: %v", orgRepo, localRepo, desc.Digest)
	adapter, err := CreateLocalRegistryAdapter()
	if err != nil {
		log.Error(err)
		return err
	}
	orgAdapter, err := CreateRegistryAdapter(proxyRegID)
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

// CreateLocalRegistryAdapter - current it only create a native adapter only,
// it should expand to other adapters for different repos
func CreateLocalRegistryAdapter() (*base.Adapter, error) {
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

func CreateRegistryAdapter(proxyRegID int64) (*native.Adapter, error) {
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
func PutManifestToLocalRepo(ctx context.Context, repo string, mfst distribution.Manifest, tag string, projectID int64) error {

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

	adapter, err := CreateLocalRegistryAdapter()
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
	if err != nil {
		log.Error(err)
		return err
	}
	return err
}

// CheckDependencies -- check all blobs used by this manifiest are ready
func CheckDependencies(ctx context.Context, man distribution.Manifest, dig string, mediaType string) []distribution.Descriptor {
	// TODO: change blob.Ctl to use HEAD method
	// TODO: CheckDependencies fails when pushing manifest list!
	descriptors := man.References()
	waitDesc := make([]distribution.Descriptor, 0)
	for _, desc := range descriptors {
		log.Infof("checking the blob depedency: %v", desc.Digest)
		exist, err := blob.Ctl.Exist(ctx, string(desc.Digest))
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

func TrimManifestList(manifest distribution.Manifest, os, arch, variant string) (distribution.Manifest, error) {
	switch v := manifest.(type) {
	case *manifestlist.DeserializedManifestList:
		trimedList := make([]manifestlist.ManifestDescriptor, 0)
		for _, m := range v.Manifests {
			if m.Platform.OS == os && m.Platform.Architecture == arch && m.Platform.Variant == variant {
				trimedList = append(trimedList, m)
			}
		}
		return manifestlist.FromDescriptors(trimedList)
	}
	return manifest, nil
}
