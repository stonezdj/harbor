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

package sbom

import (
	"context"
	"encoding/json"
	"io"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/goharbor/harbor/src/controller/artifact/processor"
	"github.com/goharbor/harbor/src/controller/artifact/processor/base"
	"github.com/goharbor/harbor/src/lib/errors"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/pkg/artifact"
)

const (
	// ArtifactTypeSBOM is the artifact type for SBOM
	ArtifactTypeSBOM = "SBOM"
	mediaType        = "application/vnd.goharbor.harbor.sbom.v1"
	notFoundMsg      = "The sbom is not found with error %v"
)

func init() {
	pc := &Processor{}
	pc.ManifestProcessor = base.NewManifestProcessor()
	if err := processor.Register(pc, mediaType); err != nil {
		log.Errorf("failed to register processor for media type %s: %v", mediaType, err)
		return
	}
}

// Processor is the processor for SBOM
type Processor struct {
	*base.ManifestProcessor
}

func (m *Processor) AbstractAddition(ctx context.Context, art *artifact.Artifact, addition string) (*processor.Addition, error) {
	man, _, err := m.RegCli.PullManifest(art.RepositoryName, art.Digest)
	if err != nil {
		return nil, errors.New(nil).WithCode(errors.NotFoundCode).WithMessage(notFoundMsg, err)
	}
	_, payload, err := man.Payload()
	if err != nil {
		return nil, errors.New(nil).WithCode(errors.NotFoundCode).WithMessage(notFoundMsg, err)
	}
	manifest := &v1.Manifest{}
	if err := json.Unmarshal(payload, manifest); err != nil {
		return nil, err
	}
	for _, layer := range manifest.Layers {
		layerDgst := layer.Digest.String()
		// SBOM artifact do have two layers, one is config, we should resolve the other one.
		if layerDgst != manifest.Config.Digest.String() {
			_, blob, err := m.RegCli.PullBlob(art.RepositoryName, layerDgst)
			if err != nil {
				return nil, errors.New(nil).WithCode(errors.NotFoundCode).WithMessage(notFoundMsg, err)
			}
			content, err := io.ReadAll(blob)
			if err != nil {
				return nil, err
			}
			blob.Close()
			return &processor.Addition{
				Content:     content,
				ContentType: mediaType,
			}, nil
		}
	}
	return nil, errors.New(nil).WithCode(errors.NotFoundCode).WithMessage("The sbom is not found")
}

func (m *Processor) GetArtifactType(_ context.Context, _ *artifact.Artifact) string {
	return ArtifactTypeSBOM
}
