package scheduling

import (
	"fmt"
	"strconv"
	"strings"
)

func cellStr(row []interface{}, idx int) string {
	if idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", row[idx]))
}

func cellStrPtr(row []interface{}, idx int) *string {
	s := cellStr(row, idx)
	if s == "" {
		return nil
	}
	return &s
}

func cellInt(row []interface{}, idx int, defaultVal int) int {
	s := cellStr(row, idx)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}
