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

package chartproxy

import (
	"context"

	"github.com/goharbor/harbor/src/pkg/chartproxy/dao"
	"github.com/goharbor/harbor/src/pkg/chartproxy/model"
)

type Manager interface {
	ListCharts(ctx context.Context) (chartList []*model.ChartVersion, err error)
	ContentDigest(ctx context.Context, repositoryName, tag string) (digest string, err error)
}

type ManagerImpl struct {
	dao dao.DAO
}

func (m ManagerImpl) ContentDigest(ctx context.Context, repositoryName, tag string) (digest string, err error) {
	return m.dao.ContentDigest(ctx, repositoryName, tag)
}

func (m ManagerImpl) ListCharts(ctx context.Context) (chartList []*model.ChartVersion, err error) {
	return m.dao.ListChart(ctx)
}

func NewManager() Manager {
	return &ManagerImpl{dao: dao.New()}
}
