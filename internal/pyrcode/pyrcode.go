package pyrcode

type PyrCode struct {
	init     string
	category string
}

const Init string = "PYR"

func (pyr *PyrCode) GeneratePyrCode() string {

	return "PYR"
}

func NewPyrCode(categoryType string, itemId string) PyrCode {
	var code PyrCode

	code.init = Init
	code.category = categoryType

	return code
}
