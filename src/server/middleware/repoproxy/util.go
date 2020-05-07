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
	"github.com/docker/distribution/registry/client"
	"github.com/docker/distribution/registry/client/auth"
	"github.com/docker/distribution/registry/client/auth/challenge"
	"github.com/docker/distribution/registry/client/transport"
	"github.com/docker/libtrust"
	"github.com/goharbor/harbor/src/controller/artifact"
	"github.com/goharbor/harbor/src/controller/blob"
	"github.com/goharbor/harbor/src/controller/repository"
	"github.com/goharbor/harbor/src/core/config"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/replication/adapter/native"
	"github.com/goharbor/harbor/src/replication/model"
	"github.com/opencontainers/go-digest"
	"net/http"
	"net/url"
)

func CreateRemoteRepository(ctx context.Context, repository string) (distribution.Repository, error) {
	regUrl, err := url.Parse("https://registry-1.docker.io")
	if err != nil {
		return nil, err
	}
	cs, err := configureAuth(ProxyConfig.Username, ProxyConfig.Password, ProxyConfig.URL)
	c := &remoteAuthChallenger{
		remoteURL: *regUrl,
		cm:        challenge.NewSimpleManager(),
		cs:        cs,
	}
	repo, _ := reference.WithName(repository)
	tkopts := auth.TokenHandlerOptions{
		Transport:   http.DefaultTransport,
		Credentials: c.credentialStore(),
		Scopes: []auth.Scope{
			auth.RepositoryScope{
				Repository: repo.Name(),
				Actions:    []string{"pull"},
			},
		},
	}
	c.tryEstablishChallenges(ctx)
	tr := transport.NewTransport(http.DefaultTransport,
		auth.NewAuthorizer(c.challengeManager(),
			auth.NewTokenHandlerWithOptions(tkopts)))
	return client.NewRepository(repo, "https://registry-1.docker.io", tr)
}

func GetTagFromRemote(ctx context.Context, repository string, tag string) (distribution.Descriptor, error) {
	desc := distribution.Descriptor{}
	r, err := CreateRemoteRepository(ctx, repository)
	if err != nil {
		return desc, err
	}
	tagService := r.Tags(ctx)
	return tagService.Get(ctx, tag)
}

func GetManifestFromRemote(ctx context.Context, repository string, tag string) (distribution.Manifest, distribution.Descriptor, error) {
	desc := distribution.Descriptor{}
	r, err := CreateRemoteRepository(ctx, repository)
	if err != nil {
		return nil, desc, err
	}
	tagService := r.Tags(ctx)
	d, err := tagService.Get(ctx, tag)
	if err != nil {
		return nil, d, err
	}
	ms, err := r.Manifests(ctx)
	if err != nil {
		return nil, d, err
	}
	man, err := ms.Get(ctx, d.Digest)
	if err != nil {
		return nil, d, err
	}
	return man, d, err
}
func GetManifestFromRemoteWithDigest(ctx context.Context, repository string, dig string) (distribution.Manifest, error) {
	r, err := CreateRemoteRepository(ctx, repository)
	if err != nil {
		return nil, err
	}
	ms, err := r.Manifests(ctx)
	if err != nil {
		return nil, err
	}
	man, err := ms.Get(ctx, digest.Digest(dig))
	if err != nil {
		return nil, err
	}
	return man, err
}

func GetBlobFromRemote(ctx context.Context, repository string, dig string) ([]byte, distribution.Descriptor, error) {
	d := distribution.Descriptor{}
	r, err := CreateRemoteRepository(ctx, repository)
	if err != nil {
		return nil, d, err
	}
	blobService := r.Blobs(ctx)
	desc, err := blobService.Stat(ctx, digest.Digest(dig))
	if err != nil {
		return nil, d, err
	}
	b, err := blobService.Get(ctx, digest.Digest(dig))

	return b, desc, err
}
func GetBlobFromLocalRepo(ctx context.Context, r distribution.Repository, dig string) ([]byte, distribution.Descriptor, error) {
	d := distribution.Descriptor{}
	blobService := r.Blobs(ctx)
	desc, err := blobService.Stat(ctx, digest.Digest(dig))
	if err != nil {
		return nil, d, err
	}
	b, err := blobService.Get(ctx, digest.Digest(dig))

	return b, desc, err
}

func CreateLocalRepository(ctx context.Context, proxyAuth ProxyAuth, repository string) (distribution.Repository, error) {
	reURLString := proxyAuth.URL
	regUrl, err := url.Parse(reURLString)

	log.Infof("username is %v, password is %v， URL is %v", proxyAuth.Username, proxyAuth.Password, proxyAuth.URL)
	cs, err := configureAuth(proxyAuth.Username, proxyAuth.Password, reURLString)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Current configure %+v\n", ProxyConfig)
	c := &remoteAuthChallenger{
		remoteURL: *regUrl,
		cm:        challenge.NewSimpleManager(),
		cs:        cs,
	}
	repo, _ := reference.WithName(repository)
	tkopts := auth.TokenHandlerOptions{
		Transport:   http.DefaultTransport,
		Credentials: c.credentialStore(),
		Scopes: []auth.Scope{
			auth.RepositoryScope{
				Repository: repo.Name(),
				Actions:    []string{"pull"},
			},
		},
	}
	c.tryEstablishChallenges(ctx)
	tr := transport.NewTransport(http.DefaultTransport,
		auth.NewAuthorizer(c.challengeManager(),
			auth.NewTokenHandlerWithOptions(tkopts)))
	return client.NewRepository(repo, reURLString, tr)
}

func GetManifestFromRepo(ctx context.Context, r distribution.Repository, tag string) (distribution.Manifest, distribution.Descriptor, error) {
	tagService := r.Tags(ctx)
	d, err := tagService.Get(ctx, tag)
	if err != nil {
		return nil, d, err
	}
	ms, err := r.Manifests(ctx)
	if err != nil {
		return nil, d, err
	}
	man, err := ms.Get(ctx, d.Digest)
	if err != nil {
		return nil, d, err
	}
	return man, d, err
}

func PutBlobToLocal(ctx context.Context, repo string, bl []byte, desc distribution.Descriptor, projID int64) error {
	log.Debug("Put bl to local registry!")
	adapter, err := CreateLocalReigstryAdapter()
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

func CreateLocalReigstryAdapter() (*native.Adapter, error) {
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

func PutManifestToLocalRepo(ctx context.Context, repo string, mfst distribution.Manifest, tag string, projectID int64) error {
	adapter, err := CreateLocalReigstryAdapter()
	if err != nil {
		log.Error(err)
		return err
	}
	mediaType, payload, err := mfst.Payload()
	if err != nil {
		log.Error(err)
		return err
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
