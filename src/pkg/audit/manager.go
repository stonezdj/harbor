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

package audit

import (
	"context"
	"strings"
	"time"

	beegorm "github.com/astaxie/beego/orm"
	"github.com/goharbor/harbor/src/lib/config"
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/lib/orm"
	"github.com/goharbor/harbor/src/lib/q"
	"github.com/goharbor/harbor/src/pkg/audit/dao"
	"github.com/goharbor/harbor/src/pkg/audit/model"
)

// Mgr is the global audit log manager instance
var Mgr = New()

// Manager is used for audit log management
type Manager interface {
	// Count returns the total count of audit logs according to the query
	Count(ctx context.Context, query *q.Query) (total int64, err error)
	// List audit logs according to the query
	List(ctx context.Context, query *q.Query) (audits []*model.AuditLog, err error)
	// Get the audit log specified by ID
	Get(ctx context.Context, id int64) (audit *model.AuditLog, err error)
	// Create the audit log
	Create(ctx context.Context, audit *model.AuditLog) (id int64, err error)
	// Delete the audit log specified by ID
	Delete(ctx context.Context, id int64) (err error)
	// StartAuditLogPurger
	StartAuditLogPurger()
}

// New returns a default implementation of Manager
func New() Manager {
	return &manager{
		dao: dao.New(),
	}
}

type manager struct {
	dao dao.DAO
}

// Count ...
func (m *manager) Count(ctx context.Context, query *q.Query) (int64, error) {
	return m.dao.Count(ctx, query)
}

// List ...
func (m *manager) List(ctx context.Context, query *q.Query) ([]*model.AuditLog, error) {
	return m.dao.List(ctx, query)
}

// Get ...
func (m *manager) Get(ctx context.Context, id int64) (*model.AuditLog, error) {
	return m.dao.Get(ctx, id)
}

// Create ...
func (m *manager) Create(ctx context.Context, audit *model.AuditLog) (int64, error) {
	if strings.EqualFold(audit.Operation, "pull") &&
		strings.EqualFold(audit.Username, "anonymous") {
		log.AL.WithField("operator", audit.Username).
			WithField("time", audit.OpTime).
			Debugf("%s :%s", audit.Operation, audit.Resource)
	} else {
		log.AL.WithField("operator", audit.Username).
			WithField("time", audit.OpTime).
			Infof("%s :%s", audit.Operation, audit.Resource)
	}
	if config.AuditLogRetentionHour(ctx) == -1 {
		return 0, nil
	}
	return m.dao.Create(ctx, audit)
}

// Purge ...
func (m *manager) Purge(ctx context.Context, retentionHours int) error {
	return m.dao.Purge(ctx, retentionHours)
}

// StartAuditLogPurger ...
func (m *manager) StartAuditLogPurger() {
	ctx := orm.NewContext(context.TODO(), beegorm.NewOrm())
	interval := config.AuditLogPurgeInterval(ctx)
	go func() {
		for {
			// TODO: create redis lock make sure only one task running
			retentionHour := config.AuditLogRetentionHour(ctx)
			if retentionHour == -1 || retentionHour == 0 {
				continue
			}
			if err := m.Purge(ctx, retentionHour); err != nil {
				log.Errorf("failed to purge audit log, error %v", err)
			}
			time.Sleep(time.Duration(interval) * time.Second)
		}
	}()
}

// Delete ...
func (m *manager) Delete(ctx context.Context, id int64) error {
	return m.dao.Delete(ctx, id)
}
