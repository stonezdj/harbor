//  Copyright Project Harbor Authors
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package handler

import (
	"context"

	"github.com/go-openapi/runtime/middleware"

	"github.com/goharbor/harbor/src/common/security"
	securityhubCtl "github.com/goharbor/harbor/src/controller/securityhub"
	"github.com/goharbor/harbor/src/lib/errors"
	"github.com/goharbor/harbor/src/pkg/scan/dao/scan"
	secHubModel "github.com/goharbor/harbor/src/pkg/securityhub/model"
	"github.com/goharbor/harbor/src/server/v2.0/models"
	"github.com/goharbor/harbor/src/server/v2.0/restapi/operations/securityhub"
)

func newSecurityAPI() *securityAPI {
	return &securityAPI{
		controller: securityhubCtl.Ctl,
	}
}

type securityAPI struct {
	BaseAPI
	controller securityhubCtl.Controller
}

func (s *securityAPI) GetSecuritySummary(ctx context.Context,
	params securityhub.GetSecuritySummaryParams) middleware.Responder {
	secCtx, ok := security.FromContext(ctx)
	if !ok {
		return s.SendError(ctx, errors.UnauthorizedError(errors.New("security context not found")))
	}
	if !secCtx.IsAuthenticated() && !secCtx.IsSysAdmin() {
		return s.SendError(ctx, errors.UnauthorizedError(nil).WithMessage(secCtx.GetUsername()))
	}
	summary, err := s.controller.SecuritySummary(ctx, 0, *params.WithDangerousCVE, *params.WithDangerousArtifact)
	if err != nil {
		return s.SendError(ctx, err)
	}
	sum := toSecuritySummaryModel(summary)
	return securityhub.NewGetSecuritySummaryOK().WithPayload(sum)
}

func toSecuritySummaryModel(summary *secHubModel.Summary) *models.SecuritySummary {
	return &models.SecuritySummary{
		CriticalCnt:        summary.CriticalCnt,
		HighCnt:            summary.HighCnt,
		MediumCnt:          summary.MediumCnt,
		LowCnt:             summary.LowCnt,
		NoneCnt:            summary.NoneCnt,
		UnknownCnt:         summary.UnknownCnt,
		FixableCnt:         summary.FixableCnt,
		TotalVuls:          summary.CriticalCnt + summary.HighCnt + summary.MediumCnt + summary.LowCnt + summary.NoneCnt + summary.UnknownCnt,
		TotalArtifact:      summary.TotalArtifactCnt,
		ScannedCnt:         summary.ScannedCnt,
		DangerousCves:      toDangerousCves(summary.DangerousCVEs),
		DangerousArtifacts: toDangerousArtifacts(summary.DangerousArtifacts),
	}
}
func toDangerousArtifacts(artifacts []*secHubModel.DangerousArtifact) []*models.DangerousArtifact {
	var result []*models.DangerousArtifact
	for _, artifact := range artifacts {
		result = append(result, &models.DangerousArtifact{
			ProjectID:      artifact.Project,
			RepositoryName: artifact.Repository,
			Digest:         artifact.Digest,
			CriticalCnt:    artifact.CriticalCnt,
			HighCnt:        artifact.HighCnt,
			MediumCnt:      artifact.MediumCnt,
		})
	}
	return result
}

func toDangerousCves(cves []*scan.VulnerabilityRecord) []*models.DangerousCVE {
	var result []*models.DangerousCVE
	for _, vul := range cves {
		result = append(result, &models.DangerousCVE{
			CVEID:       vul.CVEID,
			Package:     vul.Package,
			Version:     vul.PackageVersion,
			Severity:    vul.Severity,
			CvssScoreV3: *vul.CVE3Score,
		})
	}
	return result
}

func (s *securityAPI) ListVulnerabilities(ctx context.Context, params securityhub.ListVulnerabilitiesParams) middleware.Responder {
	secCtx, ok := security.FromContext(ctx)
	if !ok {
		return s.SendError(ctx, errors.UnauthorizedError(errors.New("security context not found")))
	}
	if !secCtx.IsSysAdmin() {
		return s.SendError(ctx, errors.UnauthorizedError(errors.New("only admin can access cve list")))
	}
	query, err := s.BuildQuery(ctx, params.Q, params.Sort, params.Page, params.PageSize)
	if err != nil {
		return s.SendError(ctx, err)
	}
	cnt, err := s.controller.CountVuls(ctx, 0, *params.TuneCount, query)
	if err != nil {
		return s.SendError(ctx, err)
	}
	vuls, err := s.controller.ListVuls(ctx, 0, query)
	if err != nil {
		return s.SendError(ctx, err)
	}
	link := s.Links(ctx, params.HTTPRequest.URL, cnt, query.PageNumber, query.PageSize).String()
	return securityhub.NewListVulnerabilitiesOK().WithPayload(toVulnerabilities(vuls)).WithLink(link).WithXTotalCount(cnt)
}

func toVulnerabilities(vuls []*secHubModel.VulnerabilityItem) []*models.VulnerabilityItem {
	result := make([]*models.VulnerabilityItem, 0)
	for _, item := range vuls {
		score := float32(0)
		if item.CVE3Score != nil {
			score = float32(*item.CVE3Score)
		}
		result = append(result, &models.VulnerabilityItem{
			Project:      item.Project,
			Repository:   item.Repository,
			Digest:       item.Digest,
			CVEID:        item.CVEID,
			Severity:     item.Severity,
			Package:      item.Package,
			Tags:         item.Tags,
			Version:      item.PackageVersion,
			FixedVersion: item.Fix,
			Desc:         item.Description,
			CvssV3Score:  score,
			URL:          item.URLs,
		})
	}
	return result
}
