package repoproxy

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseBlob(t *testing.T) {
	url := "/v2/library/hello-world/blobs/sha256:0e03bdcc26d7a9a57ef3b6f1bf1a210cff6239bff7c8cac72435984032851689"
	result := parseBlob(url)
	assert.Equal(t, "sha256:0e03bdcc26d7a9a57ef3b6f1bf1a210cff6239bff7c8cac72435984032851689", result)
	url2 := "/v2/library/hello-world/bad"
	result2 := parseBlob(url2)
	assert.Equal(t, "", result2)
}

func TestParseProject(t *testing.T) {
	url := "/v2/library/hello-world/blobs/sha256:0e03bdcc26d7a9a57ef3b6f1bf1a210cff6239bff7c8cac72435984032851689"
	result := parseProject(url)
	assert.Equal(t, "library", result)
}

func TestParseRepo(t *testing.T) {
	url := "/v2/library/hello-world/blobs/sha256:0e03bdcc26d7a9a57ef3b6f1bf1a210cff6239bff7c8cac72435984032851689"
	result := parseRepo(url)
	assert.Equal(t, "library/hello-world", result)
}