package repoproxy

import (
	"github.com/docker/distribution"
	"github.com/docker/distribution/reference"
	"net/url"
	"fmt"
	"context"
	"github.com/docker/distribution/registry/client/auth/challenge"
	"github.com/docker/distribution/registry/client/auth"
	"github.com/docker/distribution/registry/client"
	"net/http"
	"github.com/docker/distribution/registry/client/transport"
	"github.com/opencontainers/go-digest"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest"
	"github.com/docker/libtrust"
	"crypto/rand"
	"github.com/goharbor/harbor/src/lib/errors"
	"github.com/goharbor/harbor/src/controller/blob"
)

func CreateRemoteRepository(ctx context.Context, repository string) (distribution.Repository, error){
	regUrl, err:=url.Parse("https://registry-1.docker.io")
	if err != nil {
		return nil, err
	}
	cs, err := configureAuth(ProxyConfig.Username, ProxyConfig.Password, ProxyConfig.URL)
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
	return client.NewRepository(repo,"https://registry-1.docker.io", tr)
}

func GetTagFromRemote(ctx context.Context, repository string, tag string) (distribution.Descriptor, error){
	desc := distribution.Descriptor{}
	r, err := CreateRemoteRepository(ctx, repository)
	if err!= nil {
		return desc, err
	}
	tagService:=r.Tags(ctx)
	return tagService.Get(ctx, tag)
}

func GetManifestFromRemote(ctx context.Context, repository string, tag string)(distribution.Manifest, distribution.Descriptor, error){
	desc := distribution.Descriptor{}
	r, err := CreateRemoteRepository(ctx, repository)
	if err!= nil {
		return nil, desc, err
	}
	tagService:=r.Tags(ctx)
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
func GetManifestFromRemoteWithDigest(ctx context.Context, repository string, dig string)(distribution.Manifest, error){
	r, err := CreateRemoteRepository(ctx, repository)
	if err!= nil {
		return nil, err
	}
	ms, err := r.Manifests(ctx)
	if err != nil {
		return nil,  err
	}
	man, err := ms.Get(ctx, digest.Digest(dig))
	if err != nil {
		return nil,  err
	}
	return man, err
}

func GetBlobFromRemote(ctx context.Context, repository string, dig string) ([]byte, distribution.Descriptor, error){
	d := distribution.Descriptor{}
	r, err := CreateRemoteRepository(ctx, repository)
	if err != nil {
		return nil, d, err
	}
	blobService := r.Blobs(ctx)
	desc, err:=blobService.Stat(ctx, digest.Digest(dig))
	if err != nil {
		return nil, d, err
	}
	b, err:= blobService.Get(ctx, digest.Digest(dig))

	return b, desc, err
}
func GetBlobFromLocalRepo(ctx context.Context, r distribution.Repository,  dig string) ([]byte, distribution.Descriptor, error){
	d := distribution.Descriptor{}
	blobService := r.Blobs(ctx)
	desc, err:=blobService.Stat(ctx, digest.Digest(dig))
	if err != nil {
		return nil, d, err
	}
	b, err:= blobService.Get(ctx, digest.Digest(dig))

	return b, desc, err
}



func CreateLocalRepository(ctx context.Context, proxyAuth ProxyAuth, repository string) (distribution.Repository, error){
	reURLString := proxyAuth.URL
	regUrl, err:=url.Parse(reURLString)

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
	return client.NewRepository(repo,reURLString, tr)
}


func GetManifestFromRepo(ctx context.Context, r distribution.Repository, tag string)(distribution.Manifest, distribution.Descriptor, error){
	tagService:=r.Tags(ctx)
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

func PutManifestToLocal(ctx context.Context, repo string,  mfst distribution.Manifest, tag string) error {
	authCfg:=ProxyAuth{
		URL: "http://core:8080",
		Username: "admin",
		Password: "Harbor12345",
	}

	r, err := CreateLocalRepository(ctx, authCfg, repo)
	if err != nil {
		return err
	}
	return PutManifestToLocalRepo(ctx, r, mfst, tag)
}

func PutBlobToLocal(ctx context.Context, repo string, blob []byte, desc distribution.Descriptor) error {
	authCfg:=ProxyAuth{
		URL: "http://10.117.180.159:8080",
		Username: "admin",
		Password: "Harbor12345",
	}

	r, err := CreateLocalRepository(ctx, authCfg, repo)
	if err != nil {
		return err
	}
	return PutBlobToLocalRepo(ctx, r, blob, desc)
}

func PutManifestToLocalRepo(ctx context.Context, r distribution.Repository, mfst distribution.Manifest, tag string) error {
	manifestService, err:= r.Manifests(ctx)
	if err != nil {
		return err
	}
	if len(tag)>0 {
		_, err=manifestService.Put(ctx, mfst, distribution.WithTag(tag))
		return err
	}
	_, err=manifestService.Put(ctx, mfst)
	return err

}

func PutBlobToLocalRepo(ctx context.Context, r distribution.Repository, blob []byte, desc distribution.Descriptor) error{
	blobService:=r.Blobs(ctx)
	_, err:=blobService.Put(ctx, desc.MediaType, blob)
	if err!= nil {
		return err
	}
	return nil
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

type Func func(attempt int)(retry bool, err error)

const MaxRetries = 30
func Try(fn Func) error {
	var err error
	var cont bool
	attempt := 1
	for {
		cont, err = fn(attempt)
		if ! cont || err == nil {
			break
		}
		attempt++
		if attempt > MaxRetries {
			return errors.New("Max tries reached")
		}
	}
	return err
}


func CheckDependencies(ctx context.Context, man distribution.Manifest) bool {
	descriptors :=man.References()
	for _, desc := range descriptors {
		exist, err:=blob.Ctl.Exist(ctx, string(desc.Digest))
		if err!=nil {
			return false
		}
		if !exist {
			return false
		}
	}
	return true

}