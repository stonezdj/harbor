package model

// Report of the scan.
// Identified by the `digest`, `registration_uuid` and `mime_type`.
type Report struct {
	ID               int64  `orm:"pk;auto;column(id)"`
	UUID             string `orm:"unique;column(uuid)"`
	ArtifactID       int64  `orm:"column(artifact_id)"`
	RegistrationUUID string `orm:"column(registration_uuid)"`
	MimeType         string `orm:"column(mime_type)"`
	Report           string `orm:"column(report);type(json)"`
}

// TableName for Report
func (r *Report) TableName() string {
	return "sbom_report"
}
