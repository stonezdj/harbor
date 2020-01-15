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

package handler

import (
	"context"

	"github.com/go-openapi/runtime/middleware"
	"github.com/goharbor/harbor/src/api/artifact"
	"github.com/goharbor/harbor/src/pkg/q"
	"github.com/goharbor/harbor/src/server/v2.0/models"
	operation "github.com/goharbor/harbor/src/server/v2.0/restapi/operations/artifact"
)

// ArtifactAPI the api implemention of artifacts
type ArtifactAPI struct {
	BaseAPI
	Ctl artifact.Controller
}

// DeleteArtifact ...
func (api *ArtifactAPI) DeleteArtifact(ctx context.Context, params operation.DeleteArtifactParams) middleware.Responder {
	err := api.Ctl.Delete(ctx, params.ArtifactID)
	if err == nil {
		return operation.NewDeleteArtifactOK()
	} else {
		//???
	}
	return operation.NewDeleteArtifactOK()
}

// ListArtifacts ...
func (api *ArtifactAPI) ListArtifacts(ctx context.Context, params operation.ListArtifactsParams) middleware.Responder {
	query := &q.Query{
		PageNumber: int64(*params.Page),
		PageSize:   int64(*params.PageSize),
	}

	query.Keywords["ProjectID"] = params.ProjectID
	query.Keywords["RepositoryID"] = params.RepositoryID

	option := &artifact.Option{
		WithTag:        true,
		WithScanResult: true,
		WithSignature:  true,
	}
	_, alist, err := api.Ctl.List(ctx, query, option)
	if err != nil {
		//log error and return error
		operation.NewListArtifactsForbidden()
	}

	return operation.NewListArtifactsOK().WithPayload(copyArtifactList(alist))
}

func copyArtifactList(list []*artifact.Artifact) []*models.Artifact {
	artifactList := make([]*models.Artifact, 0)
	for _, a := range list {
		artifact := &models.Artifact{
			Digest:    a.Digest,
			ID:        a.ID,
			MediaType: a.MediaType,
			Size:      a.Size,
			Type:      a.Type,
			//UploadTime: a.PushTime,
		}
		artifactList = append(artifactList, artifact)
	}
	return artifactList
}

// ReadArtifact ...
func (api *ArtifactAPI) ReadArtifact(ctx context.Context, params operation.ReadArtifactParams) middleware.Responder {
	return operation.NewReadArtifactOK()
}

// NewArtifactAPI returns API of artifacts
func NewArtifactAPI() *ArtifactAPI {
	return &ArtifactAPI{}
}
