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
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/reference"
	"github.com/docker/libtrust"
	"github.com/goharbor/harbor/src/common"
	"github.com/goharbor/harbor/src/controller/artifact"
	"github.com/goharbor/harbor/src/controller/blob"
	"github.com/goharbor/harbor/src/controller/repository"
	"github.com/goharbor/harbor/src/core/config"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/replication/adapter/native"
	"github.com/goharbor/harbor/src/replication/model"
	"github.com/opencontainers/go-digest"
	"io/ioutil"
	"strings"
)

func GetManifestFromTarget(ctx context.Context, repository string, tag string) (distribution.Manifest, distribution.Descriptor, error) {
	desc := distribution.Descriptor{}
	adapter, err := CreateRemoteRegistryAdapter(ProxyConfig)
	if err != nil {
		log.Error(err)
		return nil, desc, nil
	}
	man, dig, err := adapter.PullManifest(repository, tag)
	desc.Digest = digest.Digest(dig)
	return man, desc, nil
}

func GetManifestFromTargetWithDigest(ctx context.Context, repository string, dig string) (distribution.Manifest, error) {
	adapter, err := CreateRemoteRegistryAdapter(ProxyConfig)
	man, dig, err := adapter.PullManifest(repository, dig) //if tag is not provided, the digest also works
	return man, err
}

func GetBlobFromTarget(ctx context.Context, repository string, dig string) ([]byte, distribution.Descriptor, error) {
	d := distribution.Descriptor{}
	adapter, err := CreateRemoteRegistryAdapter(ProxyConfig)
	if err != nil {
		return nil, d, err
	}

	desc, bReader, err := adapter.PullBlob(repository, dig)
	if err != nil {
		log.Error(err)
	}
	blob, err := ioutil.ReadAll(bReader)
	defer bReader.Close()
	if err != nil {
		log.Error(err)
	}
	if string(desc.Digest) != dig {
		log.Errorf("origin dig:%v actual: %v", dig, string(desc.Digest))
	}
	d.Size = desc.Size
	d.MediaType = desc.MediaType
	d.Digest = digest.Digest(dig)

	return blob, d, err
}

func PutBlobToLocal(ctx context.Context, repo string, bl []byte, desc distribution.Descriptor, projID int64) error {
	log.Debugf("Put bl to local registry!, digest: %v", desc.Digest)
	adapter, err := CreateLocalRegistryAdapter()
	if err != nil {
		log.Error(err)
		return err
	}
	err = adapter.PushBlob(repo, string(desc.Digest), desc.Size, bytes.NewReader(bl))
	if err == nil {
		blobID, err := blob.Ctl.Ensure(ctx, string(desc.Digest), desc.MediaType, desc.Size)
		if err != nil {
			log.Error(err)
		}
		err = blob.Ctl.AssociateWithProjectByID(ctx, blobID, projID)
		if err != nil {
			log.Error(err)
		}
	}
	return err
}

// CreateLocalRegistryAdapter - current it only create a native adapter only,
// it should expand to other adapters for different repos
func CreateLocalRegistryAdapter() (*native.Adapter, error) {
	username, password := config.RegistryCredential()
	registryURL, err := config.RegistryURL()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	reg := &model.Registry{
		URL: registryURL,
		Credential: &model.Credential{
			Type:         model.CredentialTypeBasic,
			AccessKey:    username,
			AccessSecret: password,
		},
	}
	return native.NewAdapter(reg), nil
}

func CreateRemoteRegistryAdapter(proxyAuth *ProxyAuth) (*native.Adapter, error) {
	reg := &model.Registry{
		URL: proxyAuth.URL,
		Credential: &model.Credential{
			Type:         model.CredentialTypeBasic,
			AccessKey:    proxyAuth.Username,
			AccessSecret: proxyAuth.Password,
		},
		Insecure: true,
	}
	return native.NewAdapter(reg), nil
}

func PutManifestToLocalRepo(ctx context.Context, repo string, mfst distribution.Manifest, tag string, projectID int64) error {
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
	log.Infof("Pushing manifest to repo: %v, tag:%v", repo, tag)
	if tag == "" {
		tag = "latest"
	}
	dig, err := adapter.PushManifest(repo, tag, mediaType, payload)
	if err != nil {
		log.Error(err)
		return err
	}
	_, _, err = repository.Ctl.Ensure(ctx, repo)
	if err != nil {
		log.Error(err)
	}
	_, _, err = artifact.Ctl.Ensure(ctx, repo, dig, tag)
	if err != nil {
		log.Error(err)
	}
	blobDigests := make([]string, 0)
	for _, des := range mfst.References() {
		blobDigests = append(blobDigests, string(des.Digest))
	}
	blobDigests = append(blobDigests, dig)

	log.Debugf("Blob digest %+v, %v", blobDigests, dig)
	blobID, err := blob.Ctl.Ensure(ctx, dig, mediaType, int64(len(payload)))
	blob.Ctl.AssociateWithProjectByID(ctx, blobID, projectID)

	if err != nil {
		log.Error("failed to create blob for manifest!")
	}
	err = blob.Ctl.AssociateWithArtifact(ctx, blobDigests, dig)

	if err != nil {
		log.Errorf("Failed to associate blob with artifact:%v", err)
	}

	return err
}

func newRandomBlob(size int) (digest.Digest, []byte) {
	b := make([]byte, size)
	if n, err := rand.Read(b); err != nil {
		panic(err)
	} else if n != size {
		panic("unable to read enough bytes")
	}

	return digest.FromBytes(b), b
}

func newRandomSchemaV1Manifest(name reference.Named, tag string, blobCount int) (*schema1.SignedManifest, digest.Digest, []byte) {
	blobs := make([]schema1.FSLayer, blobCount)
	history := make([]schema1.History, blobCount)

	for i := 0; i < blobCount; i++ {
		dgst, blob := newRandomBlob((i % 5) * 16)

		blobs[i] = schema1.FSLayer{BlobSum: dgst}
		history[i] = schema1.History{V1Compatibility: fmt.Sprintf("{\"Hex\": \"%x\"}", blob)}
	}

	m := schema1.Manifest{
		Name:         name.String(),
		Tag:          tag,
		Architecture: "x86",
		FSLayers:     blobs,
		History:      history,
		Versioned: manifest.Versioned{
			SchemaVersion: 1,
		},
	}

	pk, err := libtrust.GenerateECP256PrivateKey()
	if err != nil {
		panic(err)
	}

	sm, err := schema1.Sign(&m, pk)
	if err != nil {
		panic(err)
	}

	return sm, digest.FromBytes(sm.Canonical), sm.Canonical
}

func CheckDependencies(ctx context.Context, man distribution.Manifest, dig string) bool {
	descriptors := man.References()
	for _, desc := range descriptors {
		log.Infof("checking the blob depedency: %v", desc.Digest)
		exist, err := blob.Ctl.Exist(ctx, string(desc.Digest))
		if err != nil {
			log.Info("Check dependency failed!")
			return false
		}
		if !exist {
			log.Info("Check dependency failed!")
			return false
		}
	}

	log.Info("Check dependency success!")
	return true

}

func TrimProxyPrefix(repo string) string {
	if strings.HasPrefix(repo, common.ProxyNamespacePrefix) {
		return strings.TrimPrefix(repo, common.ProxyNamespacePrefix)
	}
	return repo
}
