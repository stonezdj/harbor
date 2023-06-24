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
	securityCtl "github.com/goharbor/harbor/src/controller/securitysummary"
	"github.com/goharbor/harbor/src/lib/errors"
	"github.com/goharbor/harbor/src/pkg/scan/dao/scan"
	"github.com/goharbor/harbor/src/pkg/scan/scanner"
	"github.com/goharbor/harbor/src/server/v2.0/models"
	"github.com/goharbor/harbor/src/server/v2.0/restapi/operations/securitysummary"
)

func newSecurityAPI() *securityAPI {
	return &securityAPI{
		securityCtl: securityCtl.Ctl,
		scannerMgr:  scanner.New(),
	}
}

type securityAPI struct {
	BaseAPI
	securityCtl securityCtl.Controller
	scannerMgr  scanner.Manager
}

func (s *securityAPI) GetCVEList(ctx context.Context, params securitysummary.GetCVEListParams) middleware.Responder {
	secCtx, ok := security.FromContext(ctx)
	if !ok {
		return s.SendError(ctx, errors.UnauthorizedError(errors.New("security context not found")))
	}
	if !secCtx.IsAuthenticated() || !secCtx.IsSysAdmin() {
		return s.SendError(ctx, errors.UnauthorizedError(errors.New("only admin can access cve list")))
	}
	query, err := s.BuildQuery(ctx, params.Q, params.Sort, params.Page, params.PageSize)
	defaultReg, err := s.scannerMgr.GetDefault(ctx)
	if err != nil {
		return s.SendError(ctx, err)
	}
	cnt, err := s.securityCtl.GetTotalCVEs(ctx, query, defaultReg.UUID)
	if err != nil {
		return s.SendError(ctx, err)
	}
	cveList, err := s.securityCtl.GetCVEs(ctx, query, defaultReg.UUID)
	link := s.Links(ctx, params.HTTPRequest.URL, cnt, query.PageNumber, query.PageSize).String()
	return securitysummary.NewGetCVEListOK().WithPayload(toCVERecord(cveList)).WithXTotalCount(cnt).WithLink(link)
}

func toCVERecord(cveList []*scan.VulnerabilityRecord) []*models.CVERecord {
	var result []*models.CVERecord
	for _, cve := range cveList {
		score := 0.0
		if cve.CVE3Score != nil {
			score = *cve.CVE3Score
		}
		result = append(result, &models.CVERecord{
			CVEID:       cve.CVEID,
			CvssV3Score: score,
			CweIds:      cve.CVEID,
			URL:         cve.URLs,
			Severity:    cve.Severity,
			Description: cve.Description,
		})
	}
	return result
}
