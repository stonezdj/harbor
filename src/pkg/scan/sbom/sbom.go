// Copyright Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sbom

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/goharbor/harbor/src/common"
	"github.com/goharbor/harbor/src/lib/config"
	scanModel "github.com/goharbor/harbor/src/pkg/scan/dao/scan"
	sbom "github.com/goharbor/harbor/src/pkg/scan/sbom/model"

	"github.com/goharbor/harbor/src/common/rbac"
	"github.com/goharbor/harbor/src/jobservice/job"
	"github.com/goharbor/harbor/src/pkg/permission/types"
	"github.com/goharbor/harbor/src/pkg/robot/model"
	"github.com/goharbor/harbor/src/pkg/scan"

	v1 "github.com/goharbor/harbor/src/pkg/scan/rest/v1"
	"github.com/goharbor/harbor/src/pkg/scan/vuln"
)

const sbomMimeType = "application/vnd.goharbor.harbor.sbom.v1"

func init() {
	scan.RegisterScanHanlder(v1.ScanTypeSbom, &scanHandler{})
}

// ScanHandler defines the Handler to generate sbom
type scanHandler struct {
}

func (v *scanHandler) RequestProducesMineTypes() []string {
	return []string{v1.MimeTypeSBOMReport}
}

func (v *scanHandler) RequestParameters() map[string]interface{} {
	return map[string]interface{}{"sbom_media_types": []string{"application/spdx+json"}}
}

// ReportURLParameter defines the parameters for scan report url
func (v *scanHandler) ReportURLParameter(_ *v1.ScanRequest) (string, error) {
	return fmt.Sprintf("sbom_media_type=%s", url.QueryEscape("application/spdx+json")), nil
}

// RequiredPermissions defines the permission used by the scan robot account
func (v *scanHandler) RequiredPermissions() []*types.Policy {
	return []*types.Policy{
		{
			Resource: rbac.ResourceRepository,
			Action:   rbac.ActionPull,
		},
		{
			Resource: rbac.ResourceRepository,
			Action:   rbac.ActionScannerPull,
		},
		{
			Resource: rbac.ResourceRepository,
			Action:   rbac.ActionPush,
		},
	}
}

// PostScan defines task specific operations after the scan is complete
func (v *scanHandler) PostScan(ctx job.Context, sr *v1.ScanRequest, _ *scanModel.Report, rawReport string, startTime time.Time, robot *model.Robot) (string, error) {
	myLogger := ctx.GetLogger()
	rpt := vuln.Report{}
	err := json.Unmarshal([]byte(rawReport), &rpt)
	if err != nil {
		return "", err
	}
	sbomContent, err := json.Marshal(rpt.SBOM)
	myLogger.Infof("sbom content is %v", string(sbomContent))
	if err != nil {
		return "", err
	}
	scanRep := v1.ScanRequest{
		Registry: sr.Registry,
		Artifact: sr.Artifact,
	}
	// the registry server url is core by default, need to replace it with real registry server url
	scanRep.Registry.URL = getRegistryServer(ctx)
	myLogger.Infof("Pushing accessory artifact to %s/%s", scanRep.Registry.URL, scanRep.Artifact.Repository)
	dgst, err := scan.GenAccessoryArt(scanRep, sbomContent, map[string]string{}, sbomMimeType, robot)
	if err != nil {
		myLogger.Errorf("error when create accessory from image %v", err)
		return "", err
	}
	// store digest in the report field, it is used in the sbom status summary and makeReportPlaceholder to delete previous sbom
	myLogger.Infof("acccessory image digest is %v", dgst)
	endTime := time.Now()
	sbomSummary := sbom.Summary{}
	sbomSummary[sbom.StartTime] = startTime
	sbomSummary[sbom.EndTime] = endTime
	sbomSummary[sbom.Duration] = int64(endTime.Sub(startTime).Seconds())
	sbomSummary[sbom.ScanStatus] = "Success"
	sbomSummary[sbom.SBOMRepository] = sr.Artifact.Repository
	sbomSummary[sbom.SBOMDigest] = dgst
	rep, err := json.Marshal(sbomSummary)
	if err != nil {
		return "", err
	}
	return string(rep), nil
}

// extract server name from config
func getRegistryServer(ctx job.Context) string {
	cfgMgr, ok := config.FromContext(ctx.SystemContext())
	myLogger := ctx.GetLogger()

	if ok {
		extURL := cfgMgr.Get(ctx.SystemContext(), common.ExtEndpoint).GetString()
		myLogger.Infof("The external url is %v", extURL)
		server := strings.TrimPrefix(extURL, "https://")
		server = strings.TrimPrefix(server, "http://")
		myLogger.Infof("The server is %v", server)
		return server
	}
	myLogger.Error("empty registry server!")
	return ""
}
