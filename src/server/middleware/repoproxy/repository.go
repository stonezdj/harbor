package repoproxy

import (
	"net/http"
	"github.com/docker/distribution"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/distribution/registry/client/auth"
	"github.com/docker/distribution/registry/client/auth/challenge"
	"net/url"
	"sync"
	"context"
	"strings"
	"github.com/docker/distribution/registry/api/v2"
	"github.com/opencontainers/go-digest"
	"github.com/docker/distribution/registry/api/errcode"
	"io"
	"io/ioutil"
	"encoding/json"
	"bytes"
)

const challengeHeader = "Docker-Distribution-Api-Version"

type RemoteRepository struct {
	tr http.RoundTripper
	cf ProxyAuth
	client http.Client
	repo distribution.Repository
}

// authChallenger encapsulates a request to the upstream to establish credential challenges
type authChallenger interface {
	tryEstablishChallenges(context.Context) error
	challengeManager() challenge.Manager
	credentialStore() auth.CredentialStore
}

type remoteAuthChallenger struct {
	remoteURL url.URL
	sync.Mutex
	cm challenge.Manager
	cs auth.CredentialStore
}

func (r *remoteAuthChallenger) credentialStore() auth.CredentialStore {
	return r.cs
}

func (r *remoteAuthChallenger) challengeManager() challenge.Manager {
	return r.cm
}

func ping(manager challenge.Manager, endpoint, versionHeader string) error {
	resp, err := http.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return manager.AddResponse(resp)
}

// tryEstablishChallenges will attempt to get a challenge type for the upstream if none currently exist
func (r *remoteAuthChallenger) tryEstablishChallenges(ctx context.Context) error {
	r.Lock()
	defer r.Unlock()

	remoteURL := r.remoteURL
	remoteURL.Path = "/v2/"
	challenges, err := r.cm.GetChallenges(remoteURL)
	if err != nil {
		return err
	}

	if len(challenges) > 0 {
		return nil
	}

	// establish challenge type with upstream
	if err := ping(r.cm, remoteURL.String(), challengeHeader); err != nil {
		return err
	}

	return nil
}
type userpass struct {
	username string
	password string
}
type credentials struct {
	creds map[string]userpass
}
func (c credentials) Basic(u *url.URL) (string, string) {
	up := c.creds[u.String()]

	return up.username, up.password
}

func (c credentials) RefreshToken(u *url.URL, service string) string {
	return ""
}

func (c credentials) SetRefreshToken(u *url.URL, service, token string) {
}
// configureAuth stores credentials for challenge responses
func configureAuth(username, password, remoteURL string) (auth.CredentialStore, error) {
	creds := map[string]userpass{}

	authURLs, err := getAuthURLs(remoteURL)
	if err != nil {
		return nil, err
	}

	for _, url := range authURLs {
		creds[url] = userpass{
			username: username,
			password: password,
		}
	}

	return credentials{creds: creds}, nil
}

func getAuthURLs(remoteURL string) ([]string, error) {
	authURLs := []string{}

	resp, err := http.Get(remoteURL + "/v2/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	for _, c := range challenge.ResponseChallenges(resp) {
		if strings.EqualFold(c.Scheme, "bearer") {
			authURLs = append(authURLs, c.Parameters["realm"])
		}
	}

	return authURLs, nil
}

type manifests struct {
	name   reference.Named
	ub     *v2.URLBuilder
	client *http.Client
	etags  map[string]string
}

func (ms *manifests) Exists(ctx context.Context, dgst digest.Digest) (bool, error) {
	ref, err := reference.WithDigest(ms.name, dgst)
	if err != nil {
		return false, err
	}
	u, err := ms.ub.BuildManifestURL(ref)
	if err != nil {
		return false, err
	}

	resp, err := ms.client.Head(u)
	if err != nil {
		return false, err
	}

	if SuccessStatus(resp.StatusCode) {
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, HandleErrorResponse(resp)
}
type contentDigestOption struct{ digest *digest.Digest }

func (o contentDigestOption) Apply(ms distribution.ManifestService) error {
	return nil
}
func (ms *manifests) Get(ctx context.Context, dgst digest.Digest, options ...distribution.ManifestServiceOption) (distribution.Manifest, error) {
	var (
		digestOrTag string
		ref         reference.Named
		err         error
		contentDgst *digest.Digest
		mediaTypes  []string
	)

	for _, option := range options {
		switch opt := option.(type) {
		case distribution.WithTagOption:
			digestOrTag = opt.Tag
			ref, err = reference.WithTag(ms.name, opt.Tag)
			if err != nil {
				return nil, err
			}
		case contentDigestOption:
			contentDgst = opt.digest
		case distribution.WithManifestMediaTypesOption:
			mediaTypes = opt.MediaTypes
		default:
			err := option.Apply(ms)
			if err != nil {
				return nil, err
			}
		}
	}

	if digestOrTag == "" {
		digestOrTag = dgst.String()
		ref, err = reference.WithDigest(ms.name, dgst)
		if err != nil {
			return nil, err
		}
	}

	if len(mediaTypes) == 0 {
		mediaTypes = distribution.ManifestMediaTypes()
	}

	u, err := ms.ub.BuildManifestURL(ref)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	for _, t := range mediaTypes {
		req.Header.Add("Accept", t)
	}

	if _, ok := ms.etags[digestOrTag]; ok {
		req.Header.Set("If-None-Match", ms.etags[digestOrTag])
	}

	resp, err := ms.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return nil, distribution.ErrManifestNotModified
	} else if SuccessStatus(resp.StatusCode) {
		if contentDgst != nil {
			dgst, err := digest.Parse(resp.Header.Get("Docker-Content-Digest"))
			if err == nil {
				*contentDgst = dgst
			}
		}
		mt := resp.Header.Get("Content-Type")
		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return nil, err
		}
		m, _, err := distribution.UnmarshalManifest(mt, body)
		if err != nil {
			return nil, err
		}
		return m, nil
	}
	return nil, HandleErrorResponse(resp)
}

// Put puts a manifest.  A tag can be specified using an options parameter which uses some shared state to hold the
// tag name in order to build the correct upload URL.
func (ms *manifests) Put(ctx context.Context, m distribution.Manifest, options ...distribution.ManifestServiceOption) (digest.Digest, error) {
	ref := ms.name
	var tagged bool

	for _, option := range options {
		if opt, ok := option.(distribution.WithTagOption); ok {
			var err error
			ref, err = reference.WithTag(ref, opt.Tag)
			if err != nil {
				return "", err
			}
			tagged = true
		} else {
			err := option.Apply(ms)
			if err != nil {
				return "", err
			}
		}
	}
	mediaType, p, err := m.Payload()
	if err != nil {
		return "", err
	}

	if !tagged {
		// generate a canonical digest and Put by digest
		_, d, err := distribution.UnmarshalManifest(mediaType, p)
		if err != nil {
			return "", err
		}
		ref, err = reference.WithDigest(ref, d.Digest)
		if err != nil {
			return "", err
		}
	}

	manifestURL, err := ms.ub.BuildManifestURL(ref)
	if err != nil {
		return "", err
	}

	putRequest, err := http.NewRequest("PUT", manifestURL, bytes.NewReader(p))
	if err != nil {
		return "", err
	}

	putRequest.Header.Set("Content-Type", mediaType)

	resp, err := ms.client.Do(putRequest)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if SuccessStatus(resp.StatusCode) {
		dgstHeader := resp.Header.Get("Docker-Content-Digest")
		dgst, err := digest.Parse(dgstHeader)
		if err != nil {
			return "", err
		}

		return dgst, nil
	}

	return "", HandleErrorResponse(resp)
}

func (ms *manifests) Delete(ctx context.Context, dgst digest.Digest) error {
	ref, err := reference.WithDigest(ms.name, dgst)
	if err != nil {
		return err
	}
	u, err := ms.ub.BuildManifestURL(ref)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	resp, err := ms.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if SuccessStatus(resp.StatusCode) {
		return nil
	}
	return HandleErrorResponse(resp)
}

// SuccessStatus returns true if the argument is a successful HTTP response
// code (in the range 200 - 399 inclusive).
func SuccessStatus(status int) bool {
	return status >= 200 && status <= 399
}

// HandleErrorResponse returns error parsed from HTTP response for an
// unsuccessful HTTP response code (in the range 400 - 499 inclusive). An
// UnexpectedHTTPStatusError returned for response code outside of expected
// range.
func HandleErrorResponse(resp *http.Response) error {
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		// Check for OAuth errors within the `WWW-Authenticate` header first
		// See https://tools.ietf.org/html/rfc6750#section-3
		for _, c := range challenge.ResponseChallenges(resp) {
			if c.Scheme == "bearer" {
				var err errcode.Error
				// codes defined at https://tools.ietf.org/html/rfc6750#section-3.1
				switch c.Parameters["error"] {
				case "invalid_token":
					err.Code = errcode.ErrorCodeUnauthorized
				case "insufficient_scope":
					err.Code = errcode.ErrorCodeDenied
				default:
					continue
				}
				if description := c.Parameters["error_description"]; description != "" {
					err.Message = description
				} else {
					err.Message = err.Code.Message()
				}

				return mergeErrors(err, parseHTTPErrorResponse(resp.StatusCode, resp.Body))
			}
		}
		err := parseHTTPErrorResponse(resp.StatusCode, resp.Body)
		if uErr, ok := err.(*client.UnexpectedHTTPResponseError); ok && resp.StatusCode == 401 {
			return errcode.ErrorCodeUnauthorized.WithDetail(uErr.Response)
		}
		return err
	}
	return &client.UnexpectedHTTPStatusError{Status: resp.Status}
}
func mergeErrors(err1, err2 error) error {
	return errcode.Errors(append(makeErrorList(err1), makeErrorList(err2)...))
}
func makeErrorList(err error) []error {
	if errL, ok := err.(errcode.Errors); ok {
		return []error(errL)
	}
	return []error{err}
}
func parseHTTPErrorResponse(statusCode int, r io.Reader) error {
	var errors errcode.Errors
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	// For backward compatibility, handle irregularly formatted
	// messages that contain a "details" field.
	var detailsErr struct {
		Details string `json:"details"`
	}
	err = json.Unmarshal(body, &detailsErr)
	if err == nil && detailsErr.Details != "" {
		switch statusCode {
		case http.StatusUnauthorized:
			return errcode.ErrorCodeUnauthorized.WithMessage(detailsErr.Details)
		case http.StatusTooManyRequests:
			return errcode.ErrorCodeTooManyRequests.WithMessage(detailsErr.Details)
		default:
			return errcode.ErrorCodeUnknown.WithMessage(detailsErr.Details)
		}
	}

	if err := json.Unmarshal(body, &errors); err != nil {
		return &client.UnexpectedHTTPResponseError{
			ParseErr:   err,
			StatusCode: statusCode,
			Response:   body,
		}
	}

	if len(errors) == 0 {
		// If there was no error specified in the body, return
		// UnexpectedHTTPResponseError.
		return &client.UnexpectedHTTPResponseError{
			ParseErr:   client.ErrNoErrorsInBody,
			StatusCode: statusCode,
			Response:   body,
		}
	}

	return errors
}
//func NewRepository(){
//	c := remoteAuthChallenger{}
//	tkopts := auth.TokenHandlerOptions{
//		Transport:   http.DefaultTransport,
//		Credentials: c.credentialStore(),
//		Scopes: []auth.Scope{
//			auth.RepositoryScope{
//				Repository: name.Name(),
//				Actions:    []string{"pull"},
//			},
//		},
//		Logger:
//	}
//
//	repo, _ := reference.WithName("test.example.com/repo1")
//	tr := transport.NewTransport(http.DefaultTransport,
//		auth.NewAuthorizer(c.challengeManager(),
//			auth.NewTokenHandlerWithOptions(tkopts)))
//	remoteRepo, err := client.NewRepository(repo,"https://registry-1.docker.io", tr)
//}