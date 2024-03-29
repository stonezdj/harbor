package sbom

import (
	"github.com/goharbor/harbor/src/pkg/accessory/model"
	"github.com/goharbor/harbor/src/pkg/accessory/model/base"
)

// Signature signature model
type HarborSBOM struct {
	base.Default
}

// Kind gives the reference type of cosign signature.
func (c *HarborSBOM) Kind() string {
	return model.RefHard
}

// IsHard ...
func (c *HarborSBOM) IsHard() bool {
	return true
}

// New returns cosign signature
func New(data model.AccessoryData) model.Accessory {
	return &HarborSBOM{base.Default{
		Data: data,
	}}
}

func init() {
	model.Register(model.TypeHarborSBOM, New)
}
