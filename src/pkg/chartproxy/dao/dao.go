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

package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"helm.sh/helm/v3/pkg/chart"

	"github.com/goharbor/harbor/src/common/utils"
	"github.com/goharbor/harbor/src/lib/orm"
	"github.com/goharbor/harbor/src/pkg/chartproxy/model"
)

const listChartSQL = `select a.repository_name repository_name, t.name name, a.extra_attrs extra_attrs, a.push_time, a.digest
 from artifact a,
  tag t
where a.id = t.artifact_id
  and a.type = 'CHART'
order by a.repository_name asc, t.name asc`

const contentDigestSQL = `select b.digest
from artifact a,
     tag t,
     artifact_blob ab,
     blob b
where a.repository_name = ?
  and t.name = ?
  and ab.digest_af = a.digest
  and ab.digest_blob = b.digest
  and b.content_type = 'application/vnd.cncf.helm.chart.content.v1.tar+gzip' limit 1`

// DAO is the interface to access the database
type DAO interface {
	ListChart(ctx context.Context) (chartList []*model.ChartVersion, err error)
	ContentDigest(ctx context.Context, repositoryName, tag string) (digest string, err error)
}

type dao struct {
}

// ChartInfo is the struct to store the chart info
type ChartInfo struct {
	RepositoryName string    `orm:"column(repository_name)"`
	TagName        string    `orm:"column(name)"`
	ExtraAttrs     string    `orm:"column(extra_attrs)"`
	PushTime       time.Time `orm:"column(push_time)"`
	Digest         string    `orm:"column(digest)"`
}

func (d dao) ContentDigest(ctx context.Context, repositoryName, tag string) (digest string, err error) {
	o, err := orm.FromContext(ctx)
	if err != nil {
		return "", err
	}
	err = o.Raw(contentDigestSQL, repositoryName, tag).QueryRow(&digest)
	return digest, err
}

func (d dao) ListChart(ctx context.Context) (chartList []*model.ChartVersion, err error) {
	o, err := orm.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	chartInfoList := make([]*ChartInfo, 0)
	_, err = o.Raw(listChartSQL).QueryRows(&chartInfoList)
	for _, chartInfo := range chartInfoList {
		c, err := toChartVersion(chartInfo)
		if err != nil {
			continue
		}
		chartList = append(chartList, c)
		log.Printf("chartInfo.Extras:%v", chartInfo.ExtraAttrs)
	}
	return chartList, err
}

func toChartVersion(chartInfo *ChartInfo) (*model.ChartVersion, error) {
	m := chart.Metadata{}
	err := json.Unmarshal([]byte(chartInfo.ExtraAttrs), &m)
	log.Printf("m=%v", m)
	version, err := semver.NewVersion(m.Version)
	if err != nil {
		return nil, err
	}
	versionStr := fmt.Sprintf("%d.%d.%d", version.Major(), version.Minor(), version.Patch())
	m.Version = versionStr
	if err != nil {
		return nil, err
	}
	pName, repoName := utils.ParseRepository(chartInfo.RepositoryName)
	if err != nil {
		return nil, err
	}
	return &model.ChartVersion{Created: chartInfo.PushTime,
		Metadata: &m,
		Digest:   strings.TrimPrefix(chartInfo.Digest, "sha256:"),
		URLs:     []string{fmt.Sprintf("%v/charts/%v/%v.tgz", pName, chartInfo.TagName, repoName)}}, nil
}

// New returns a new DAO
func New() DAO {
	return &dao{}
}
