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

package security

import (
	"context"

	"github.com/goharbor/harbor/src/lib/q"
	"github.com/goharbor/harbor/src/pkg/scan/dao/scan"
)

type Manager interface {
	// GetCVEs returns CVEs list
	GetCVEs(ctx context.Context, query *q.Query, scannerUUID string) ([]*scan.VulnerabilityRecord, error)
	// GetTotalCVEs return the count of the CVE
	GetTotalCVEs(ctx context.Context, query *q.Query, scannerUUID string) (int64, error)
}

// NewManager ...
func NewManager() Manager {
	return &manager{
		dao: scan.NewVulnerabilityRecordDao(),
	}
}

type manager struct {
	dao scan.VulnerabilityRecordDao
}

func (m *manager) GetCVEs(ctx context.Context, query *q.Query, scannerUUID string) ([]*scan.VulnerabilityRecord, error) {
	return m.dao.ListCVEs(ctx, query, scannerUUID)
}

func (m *manager) GetTotalCVEs(ctx context.Context, query *q.Query, scannerUUID string) (int64, error) {
	return m.dao.CountCVEs(ctx, query, scannerUUID)
}
