package metadata

import (
	"strconv"
	"warehouse/pkg/models"
)

type PyrCode struct {
	init     string
	category string
	id       string
}

const Init string = "PYR"

func (pyr *PyrCode) GeneratePyrCode() string {

	return pyr.init + "-" + pyr.category + pyr.id
}

func NewPyrCode(asset *models.Asset) PyrCode {
	var code PyrCode

	code.init = Init
	code.category = asset.Category.PyrID
	code.id = strconv.Itoa(asset.ID)

	return code
}
