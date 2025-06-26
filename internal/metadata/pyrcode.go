package metadata

import (
	"strconv"
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

func NewPyrCode(pyrID string, assetID int) PyrCode {
	var code PyrCode

	code.init = Init
	code.category = pyrID
	code.id = strconv.Itoa(assetID)

	return code
}
