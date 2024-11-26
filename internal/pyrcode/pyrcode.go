package pyrcode

import "warehouse/pkg/models"

type PyrCode struct {
	init     string
	category string
}

const Init string = "PYR"

func (pyr *PyrCode) GeneratePyrCode() string {

	return "PYR"
}

func NewPyrCode(asset *models.Asset) PyrCode {
	var code PyrCode

	code.init = Init
	code.category = asset.Category.PyrID

	return code
}

func getAccessoriesCode(accessories *models.AssetAccessories) string {
	return "001"
}
