package repoproxy

import (
	"github.com/docker/distribution/testutil"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/distribution/reference"
	"net/http/httptest"
	"testing"
	"net/http"
	"time"
	"github.com/opencontainers/go-digest"
	"fmt"
	"context"
	"bytes"
	"crypto/tls"
	"github.com/docker/distribution"
)

func testServer(rrm testutil.RequestResponseMap) (string, func()) {
	h := testutil.NewHandler(rrm)
	s := httptest.NewServer(h)
	return s.URL, s.Close
}

//func newRandomBlob(size int) (digest.Digest, []byte) {
//	b := make([]byte, size)
//	if n, err := rand.Read(b); err != nil {
//		panic(err)
//	} else if n != size {
//		panic("unable to read enough bytes")
//	}
//
//	return digest.FromBytes(b), b
//}

func addTestFetch(repo string, dgst digest.Digest, content []byte, m *testutil.RequestResponseMap) {
	*m = append(*m, testutil.RequestResponseMapping{
		Request: testutil.Request{
			Method: "GET",
			Route:  "/v2/" + repo + "/blobs/" + dgst.String(),
		},
		Response: testutil.Response{
			StatusCode: http.StatusOK,
			Body:       content,
			Headers: http.Header(map[string][]string{
				"Content-Length": {fmt.Sprint(len(content))},
				"Last-Modified":  {time.Now().Add(-1 * time.Second).Format(time.ANSIC)},
			}),
		},
	})

	*m = append(*m, testutil.RequestResponseMapping{
		Request: testutil.Request{
			Method: "HEAD",
			Route:  "/v2/" + repo + "/blobs/" + dgst.String(),
		},
		Response: testutil.Response{
			StatusCode: http.StatusOK,
			Headers: http.Header(map[string][]string{
				"Content-Length": {fmt.Sprint(len(content))},
				"Last-Modified":  {time.Now().Add(-1 * time.Second).Format(time.ANSIC)},
			}),
		},
	})
}
func TestCreateRepository(t *testing.T){
	dgst, blob:= newRandomBlob(1024)
	var m testutil.RequestResponseMap
	addTestFetch("library/hello-world", dgst, blob, &m)
	e, c := testServer(m)
	defer c()
	ctx := context.Background()
	repo, _ := reference.WithName("library/hello-world")
	r, err := client.NewRepository(repo, e, nil)
	if err != nil {
		t.Fatal(err)
	}
	l := r.Blobs(ctx)

	b, err := l.Get(ctx, dgst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, blob) {
		t.Fatalf("Wrong bytes values fetched: [%d]byte != [%d]byte", len(b), len(blob))
	}

}

func TestConnectDockerhubRepository(t *testing.T){

	ctx := context.Background()
	man, _, err:= GetManifestFromRemote(ctx, "firstfloor/hello-world", "latest")
	if err!=nil {
		t.Error(err)
	}
	mediatype, content, err:=man.Payload()
	fmt.Printf("The manifest mediatype is %v, payload:%v\n", mediatype, string(content))
	for _, desc := range man.References() {
		fmt.Printf("descriptor: %v\n", desc.Digest)
		b, desc, err := GetBlobFromRemote(ctx,"firstfloor/hello-world", string(desc.Digest) )
		if err != nil {
			t.Error(err)
		}
		fmt.Printf("blob: %v\n", string(b))
		fmt.Printf("The blob descriptor is %+v\n", desc)
	}
}

func TestPullImageFromLocalRepository(t *testing.T){
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	ctx := context.Background()
	proxyAuth:=ProxyAuth{
		Username:"admin",
		Password:"Harbor12345",
		URL: "https://10.193.28.58",
	}

	r, err:= CreateLocalRepository(ctx, proxyAuth, "firstfloor/hello-world")

	if err != nil {
		t.Error(err)
		//return
	}

	man, _, err:=GetManifestFromRepo(ctx, r, "latest")

	if err!=nil {
		t.Error(err)
	}
	mediatype, content, err:=man.Payload()
	fmt.Printf("The manifest mediatype is %v, payload:%v\n", mediatype, string(content))
	for _, desc := range man.References() {
		fmt.Printf("descriptor: %v\n", desc.Digest)
		b, desc, err := GetBlobFromLocalRepo(ctx, r, string(desc.Digest) )
		if err != nil {
			t.Error(err)
		}
		fmt.Printf("blob: %v\n", string(b))
		fmt.Printf("The blob descriptor is %+v\n", desc)
	}
}
func TesBlob_PutOnLocalRepository(t *testing.T) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	ctx := context.Background()
	proxyAuth := ProxyAuth{
		Username: "admin",
		Password: "Harbor12345",
		URL:      "https://10.193.28.58",
	}

	r, err := CreateLocalRepository(ctx, proxyAuth, "firstfloor/hello-world")

	if err != nil {
		t.Error(err)
		//return
	}

	dig, bl := newRandomBlob(1024)

	//tagService := r.Tags(ctx)
	blobService := r.Blobs(ctx)

	upload, err := blobService.Create(ctx)
	if err != nil {
		t.Error(err)
	}
	n, err := upload.ReadFrom(bytes.NewReader(bl))
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(bl)) {
		t.Fatalf("Unexpected ReadFrom length: %d; expected: %d", n, len(bl))
	}
	blob, err := upload.Commit(ctx, distribution.Descriptor{
		Digest: dig,
		Size:   int64(len(bl)),
	})



	if blob.Size != int64(len(bl)) {
		t.Fatalf("Unexpected blob size: %d; expected: %d", blob.Size, len(bl))
	}

}

func TestManifestTag_OnLocalRepository(t *testing.T){
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	ctx := context.Background()
	proxyAuth := ProxyAuth{
		Username: "admin",
		Password: "Harbor12345",
		URL:      "https://10.193.28.58",
	}

	r, err := CreateLocalRepository(ctx, proxyAuth, "firstfloor/hello-world")

	if err != nil {
		t.Error(err)
		//return
	}
	ms, err := r.Manifests(ctx)
	repo, _ := reference.WithName("10.193.28.58/firstfloor/hello-world")
	m1, _, _ := newRandomSchemaV1Manifest(repo, "other", 6)
	_, err= ms.Put(ctx, m1, distribution.WithTag(m1.Tag))
	if err!= nil{
		t.Error(err)
	}
}