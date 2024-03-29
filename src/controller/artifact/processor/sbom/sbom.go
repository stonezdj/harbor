package sbom

import (
	"context"
	"encoding/json"
	"io"

	"github.com/goharbor/harbor/src/controller/artifact/processor"
	"github.com/goharbor/harbor/src/controller/artifact/processor/base"
	"github.com/goharbor/harbor/src/lib/errors"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/pkg/artifact"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	ArtifactTypeSBOM = "SBOM"
	mediaType        = "application/vnd.goharbor.harbor.sbom.v1"
)

func init() {
	pc := &Processor{}
	pc.ManifestProcessor = base.NewManifestProcessor()
	if err := processor.Register(pc, mediaType); err != nil {
		log.Errorf("failed to register processor for media type %s: %v", mediaType, err)
		return
	}
}

type Processor struct {
	*base.ManifestProcessor
}

func (m *Processor) AbstractAddition(ctx context.Context, art *artifact.Artifact, addition string) (*processor.Addition, error) {

	man, _, err := m.RegCli.PullManifest(art.RepositoryName, art.Digest)
	if err != nil {
		return nil, errors.New(nil).WithCode(errors.NotFoundCode).WithMessage("The sbom is not found with error %v", err)
	}

	_, payload, err := man.Payload()
	if err != nil {
		return nil, errors.New(nil).WithCode(errors.NotFoundCode).WithMessage("The sbom is not found with error %v", err)
	}
	manifest := &v1.Manifest{}
	if err := json.Unmarshal(payload, manifest); err != nil {
		return nil, err
	}
	for _, layer := range manifest.Layers {
		// sbom artifact do have two layers, one is config, we should resolve the other one.
		layerDgst := layer.Digest.String()
		if layerDgst != manifest.Config.Digest.String() {
			_, blob, err := m.RegCli.PullBlob(art.RepositoryName, layerDgst)
			if err != nil {
				return nil, errors.New(nil).WithCode(errors.NotFoundCode).WithMessage("The sbom is not found with error %v", err)
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
