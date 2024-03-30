package scan

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/pkg/robot/model"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type Insecure bool

// func (i Insecure) RemoteOptions() []remote.Option {
// 	tr := http.DefaultTransport.(*http.Transport).Clone()
// 	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: bool(i)}
// 	return []remote.Option{remote.WithTransport(tr)}
// }

type putOptions struct {
	Insecure
	Annotations map[string]string
	Subject     string
}

// type referrer struct {
// 	Insecure
// 	annotations     map[string]string
// 	mediaType       types.MediaType
// 	bytes           []byte
// 	targetReference name.Reference
// }

func (r *referrer) Tag(img v1.Image) (name.Reference, error) {
	digest, err := img.Digest()
	if err != nil {
		return name.Digest{}, fmt.Errorf("error getting image digest: %w", err)
	}

	tag, err := name.NewDigest(
		fmt.Sprintf("%s/%s@%s", r.targetReference.Context().RegistryStr(), r.targetReference.Context().RepositoryStr(), digest.String()),
		r.NameOptions()...,
	)
	if err != nil {
		return name.Digest{}, fmt.Errorf("error creating new digest: %w", err)
	}
	return tag, nil
}

func getBearerToken(harborURL, username, password string, repository string, command string) (string, error) {
	// ?scope=repository%3Atkg%2Ftkg-telemetry%3Apull&service=harbor-registry
	loginURL := harborURL + "/service/token?scope=repository%3A" + url.QueryEscape(repository) + "%3A" + command + "&service=harbor-registry"

	// Create an HTTP client with insecure TLS configuration
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("GET", loginURL, nil)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	req.SetBasicAuth(username, password)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check if the request was successful (status code 2xx)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Failed to retrieve token. Status Code: %d", resp.StatusCode)
	}

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse the JSON response
	var tokenResponse map[string]interface{}
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return "", err
	}

	// Extract and return the bearer token
	token, exists := tokenResponse["token"]
	if !exists {
		return "", fmt.Errorf("Token not found in the response")
	}

	return fmt.Sprintf("%s", token), nil
}

func createAccessoryForImage(content []byte, subject string, artifactMediaType string, robot *model.Robot) (string, error) {
	putOptions := putOptions{
		Insecure: true,
		Subject:  subject,
	}

	ref := referrer{
		Insecure:  true,
		bytes:     content,
		mediaType: types.MediaType(artifactMediaType),
	}
	img, err := mutate.Append(empty.Image, mutate.Addendum{
		Layer: static.NewLayer(ref.bytes, ref.mediaType),
	})
	if err != nil {
		return "", err
	}
	targetRef, err := name.ParseReference(putOptions.Subject, putOptions.NameOptions()...)
	if err != nil {
		return "", err
	}
	// remoteOpts := append(ref.RemoteOptions(), remote.WithAuthFromKeychain(authn.DefaultKeychain))
	remoteOpts := append(ref.RemoteOptions(), remote.WithAuth(&authn.Basic{Username: robot.Name, Password: robot.Secret}))
	ref.targetReference = targetRef
	targetDesc, err := remote.Head(ref.targetReference, remoteOpts...)
	if err != nil {
		return "", err
	}
	if targetDesc == nil {
		return "", fmt.Errorf("targetDesc is nil")
	}

	img = mutate.MediaType(img, ocispec.MediaTypeImageManifest)

	img = mutate.ConfigMediaType(img, ref.mediaType)
	img = mutate.Annotations(img, ref.annotations).(v1.Image)
	img = mutate.Subject(img, *targetDesc).(v1.Image)

	tag, err := ref.Tag(img)
	if err != nil {
		return "", err
	}
	digest, err := img.Digest()
	if err != nil {
		return "", err
	}
	if err := remote.Write(tag, img, remoteOpts...); err != nil {
		return "", err
	}
	man, err := img.Manifest()
	if err != nil {
		return "", err
	}
	if man != nil {
		log.Infof("the manifest is not empty, subject is %v", man.Subject)
	} else {
		log.Info("manifest is emtpy!")
	}

	return digest.String(), nil
}
