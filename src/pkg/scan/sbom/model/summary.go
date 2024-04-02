package model

const (
	// SBOMRepository ...
	SBOMRepository = "sbom_repository"
	// SBOMDigest ...
	SBOMDigest = "sbom_digest"
	// StartTime ...
	StartTime = "start_time"
	// EndTime ...
	EndTime = "end_time"
	// Duration ...
	Duration = "duration"
	// ScanStatus ...
	ScanStatus = "scan_status"
)

// Summary includes the sbom summary information
type Summary map[string]interface{}

// SBOMAccessory returns the repository and digest of the SBOM
func (s Summary) SBOMAccessory() (repo, digest string) {
	if repo, ok := s[SBOMRepository].(string); ok {
		if digest, ok := s[SBOMDigest].(string); ok {
			return repo, digest
		}
	}
	return "", ""
}
