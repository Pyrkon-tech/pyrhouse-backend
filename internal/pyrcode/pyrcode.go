package pyrcode

import (
	"strconv"
	"warehouse/pkg/models"
)

type PyrCode struct {
	init      string
	category  string
	id        string
	accessory string
}

const Init string = "PYR"

func (pyr *PyrCode) GeneratePyrCode() string {

	return pyr.init + "-" + pyr.category + pyr.id + pyr.accessory
}

func NewPyrCode(asset *models.Asset) PyrCode {
	var code PyrCode

	code.init = Init
	code.category = asset.Category.PyrID
	code.id = strconv.Itoa(asset.ID)
	code.accessory = getAccessoriesCode(asset.Accessories)

	return code
}

func getAccessoriesCode(accessories []models.AssetAccessories) string {
	code := ""
	for _, _ = range accessories {
		code += "1"
	}

	return code
}
