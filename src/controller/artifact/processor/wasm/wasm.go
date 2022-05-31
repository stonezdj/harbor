// Copyright Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wasm

import (
	"context"
	"encoding/json"
	"github.com/goharbor/harbor/src/controller/artifact/processor"
	"github.com/goharbor/harbor/src/controller/artifact/processor/base"
	"github.com/goharbor/harbor/src/lib/errors"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/pkg/artifact"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

// const definitions
const (
	// ArtifactTypeImage is the artifact type for image
	ArtifactTypeWASM        = "WASM"
	AdditionTypeBuildHistory = "BUILD_HISTORY"
	mediaType        = "application/vnd.wasm.config.v1+json"
)

func init() {
	pc := &manifestV2Processor{}
	pc.ManifestProcessor = base.NewManifestProcessor()
	mediaTypes := []string{
		mediaType,
	}
	if err := processor.Register(pc, mediaTypes...); err != nil {
		log.Errorf("failed to register processor for media type %v: %v", mediaTypes, err)
		return
	}
}

// manifestV2Processor processes image with OCI manifest and docker v2 manifest
type manifestV2Processor struct {
	*base.ManifestProcessor
}

func (m *manifestV2Processor) AbstractMetadata(ctx context.Context, art *artifact.Artifact, manifest_body []byte) error {

	art.ExtraAttrs = map[string]interface{}{}
	art.ExtraAttrs["ArtifactType"] = "WebAssembly"

	manifest := &v1.Manifest{}
	if err := json.Unmarshal(manifest_body, manifest); err != nil {
		return err
	}

	if manifest.Annotations["module.wasm.image/variant"]=="compat" || manifest.Annotations["run.oci.handler"]=="wasm" {

		// for annotation way
		config := &v1.Image{}
		if err := m.UnmarshalConfig(ctx, art.RepositoryName, manifest_body, config); err != nil {
			return err
		}
		art.ExtraAttrs["manifest.config.mediaType"] = manifest.Config.MediaType
		if art.ExtraAttrs == nil {
			art.ExtraAttrs = map[string]interface{}{}
		}
		art.ExtraAttrs["created"] = config.Created
		art.ExtraAttrs["architecture"] = config.Architecture
		art.ExtraAttrs["os"] = config.OS
		art.ExtraAttrs["config"] = config.Config
		// if the author is null, try to get it from labels:
		// https://docs.docker.com/engine/reference/builder/#maintainer-deprecated
		author := config.Author
		if len(author) == 0 && len(config.Config.Labels) > 0 {
			author = config.Config.Labels["maintainer"]
		}
		art.ExtraAttrs["author"] = author
	}else {

		// for wasm-to-oci way
		art.ExtraAttrs["manifest.config.mediaType"] = mediaType
		art.ExtraAttrs["manifest.layers.mediaType"] = manifest.Layers[0].MediaType
		art.ExtraAttrs["org.opencontainers.image.title"] = manifest.Layers[0].Annotations["org.opencontainers.image.title"]
	}
	return nil
}

func (m *manifestV2Processor) AbstractAddition(ctx context.Context, artifact *artifact.Artifact, addition string) (*processor.Addition, error) {

	if addition != AdditionTypeBuildHistory {
		return nil, errors.New(nil).WithCode(errors.BadRequestCode).
			WithMessage("addition %s isn't supported for %s(manifest version 2)", addition, ArtifactTypeWASM)
	}

	mani, _, err := m.RegCli.PullManifest(artifact.RepositoryName, artifact.Digest)
	if err != nil {
		return nil, err
	}
	_, content, err := mani.Payload()
	if err != nil {
		return nil, err
	}
	config := &v1.Image{}
	if err = m.ManifestProcessor.UnmarshalConfig(ctx, artifact.RepositoryName, content, config); err != nil {
		return nil, err
	}
	content, err = json.Marshal(config.History)
	if err != nil {
		return nil, err
	}
	return &processor.Addition{
		Content:     content,
		ContentType: "application/json; charset=utf-8",
	}, nil
}

func (m *manifestV2Processor) GetArtifactType(ctx context.Context, artifact *artifact.Artifact) string {
	return ArtifactTypeWASM
}

func (m *manifestV2Processor) ListAdditionTypes(ctx context.Context, artifact *artifact.Artifact) []string {
	return []string{AdditionTypeBuildHistory}
}
