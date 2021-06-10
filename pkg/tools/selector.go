package tools

import (
	"strings"
)

func OsSelect(os string, osSelector []string) bool {
	if len(osSelector) == 0 {
		return true
	}

	for _, v := range osSelector {
		if strings.EqualFold(v, os) {
			return true
		}
	}

	return false
}

func ArchSelect(arch string, archSelector []string) bool {
	if len(archSelector) == 0 {
		return true
	}

	for _, v := range archSelector {
		if strings.EqualFold(v, arch) {
			return true
		}
	}

	return false
}
