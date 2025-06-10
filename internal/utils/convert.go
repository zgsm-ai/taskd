package utils

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ConvertMemoryToGB converts memory size in different units to integer in GB
// Supported units: G/GB, M/MB, K/KB, Mi, Gi, Ti
func ConvertMemoryToGB(memoryStr string) (int, error) {
	if k, err := strconv.Atoi(memoryStr); err == nil {
		return k, nil
	}

	// Remove spaces and convert to uppercase
	memoryStr = strings.TrimSpace(strings.ToUpper(memoryStr))

	// Extract number part and unit part
	var numStr strings.Builder
	var unitStr strings.Builder

	for _, char := range memoryStr {
		if unicode.IsDigit(char) {
			numStr.WriteRune(char)
		} else {
			unitStr.WriteRune(char)
		}
	}

	num, err := strconv.ParseFloat(numStr.String(), 64)
	if err != nil {
		return 0, fmt.Errorf("无效的数字格式: %v", err)
	}

	unit := unitStr.String()

	// Convert to GB based on unit
	switch unit {
	case "G", "GB":
		return int(num), nil
	case "M", "MB":
		return int(num / 1024), nil
	case "K", "KB":
		return int(num / (1024 * 1024)), nil
	case "MI":
		return int(num / 1024), nil
	case "GI":
		return int(num), nil
	case "TI":
		return int(num * 1024), nil
	default:
		return 0, fmt.Errorf("不支持的内存单位: %s", unit)
	}
}
