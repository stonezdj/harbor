package model

import (
	"time"

	beego_orm "github.com/beego/beego/v2/client/orm"
)

func init() {
	beego_orm.RegisterModel(&AuditLogExt{})
}

type AuditLogExt struct {
	ID                   int64     `orm:"pk;auto;column(id)" json:"id"`
	ProjectID            int64     `orm:"column(project_id)" json:"project_id"`
	Operation            string    `orm:"column(operation)" json:"operation"`
	OperationDescription string    `orm:"column(op_desc)" json:"operation_description"`
	OperationResult      bool      `orm:"column(op_result)" json:"operation_result"`
	ResourceType         string    `orm:"column(resource_type)"  json:"resource_type"`
	Resource             string    `orm:"column(resource)" json:"resource"`
	Username             string    `orm:"column(username)"  json:"username"`
	OpTime               time.Time `orm:"column(op_time)" json:"op_time" sort:"default:desc"`
	Payload              string    `orm:"column(payload)" json:"payload"`
}

// TableName for audit log
func (a *AuditLogExt) TableName() string {
	return "audit_log_ext"
}
