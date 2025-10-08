//go:build windows

package machine

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/valentin-kaiser/go-core/apperror"
)

// collectHardwareIdentifiersWithOptions gathers Windows-specific hardware identifiers based on generator config
func collectHardwareIdentifiersWithOptions(g *generator) ([]string, error) {
	if g == nil {
		return nil, apperror.NewError("generator cannot be nil")
	}

	var collectors []func() []string

	if g.includeCPU {
		collectors = append(collectors, func() []string { return collectIfValid(getWindowsCPUID, "cpu:") })
	}
	if g.includeMotherboard {
		collectors = append(collectors, func() []string { return collectIfValid(getWindowsMotherboardSerial, "mb:") })
	}
	if g.includeSystemUUID {
		collectors = append(collectors, func() []string { return collectIfValid(getWindowsSystemUUID, "uuid:") })
	}
	if g.includeMAC {
		collectors = append(collectors, func() []string { return collectMultipleIfValid(getWindowsMACAddresses, "mac:") })
	}
	if g.includeDisk {
		collectors = append(collectors, func() []string { return collectMultipleIfValid(getWindowsDiskSerials, "disk:") })
	}

	return collectFromAll(collectors), nil
}

// collectIfValid adds single identifier with prefix if valid
func collectIfValid(getValue func() (string, error), prefix string) []string {
	value, err := getValue()
	switch {
	case err != nil, value == "":
		return nil
	default:
		return []string{prefix + value}
	}
}

// collectMultipleIfValid adds multiple identifiers with prefix if valid
func collectMultipleIfValid(getValues func() ([]string, error), prefix string) []string {
	values, err := getValues()
	switch {
	case err != nil, len(values) == 0:
		return nil
	default:
		return prefixSlice(values, prefix)
	}
}

// prefixSlice adds prefix to each string in slice
func prefixSlice(values []string, prefix string) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = prefix + value
	}
	return result
}

// collectFromAll runs all collectors and combines results
func collectFromAll(collectors []func() []string) []string {
	var identifiers []string
	for _, collector := range collectors {
		identifiers = append(identifiers, collector()...)
	}
	return identifiers
}

// parseWmicValue extracts value from wmic output with given prefix
func parseWmicValue(output, prefix string) (string, error) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			value := strings.TrimSpace(strings.TrimPrefix(line, prefix))
			switch value {
			case "", "To be filled by O.E.M.":
				continue
			default:
				return value, nil
			}
		}
	}
	return "", fmt.Errorf("value with prefix %s not found", prefix)
}

// parseWmicMultipleValues extracts all values from wmic output with given prefix
func parseWmicMultipleValues(output, prefix string) []string {
	var values []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			value := strings.TrimSpace(strings.TrimPrefix(line, prefix))
			switch value {
			case "", "To be filled by O.E.M.":
				// Skip empty or placeholder values
			default:
				values = append(values, value)
			}
		}
	}
	return values
}

// getWindowsCPUID retrieves CPU processor ID using wmic
func getWindowsCPUID() (string, error) {
	cmd := exec.Command("wmic", "cpu", "get", "ProcessorId", "/value")
	output, err := cmd.Output()
	switch err {
	case nil:
		return parseWmicValue(string(output), "ProcessorId=")
	default:
		return "", err
	}
}

// getWindowsMotherboardSerial retrieves motherboard serial number using wmic
func getWindowsMotherboardSerial() (string, error) {
	cmd := exec.Command("wmic", "baseboard", "get", "SerialNumber", "/value")
	output, err := cmd.Output()
	switch err {
	case nil:
		return parseWmicValue(string(output), "SerialNumber=")
	default:
		return "", err
	}
}

// getWindowsSystemUUID retrieves system UUID using wmic
func getWindowsSystemUUID() (string, error) {
	cmd := exec.Command("wmic", "csproduct", "get", "UUID", "/value")
	output, err := cmd.Output()
	switch err {
	case nil:
		return parseWmicValue(string(output), "UUID=")
	default:
		return "", err
	}
}

// getWindowsMACAddresses retrieves MAC addresses using wmic
func getWindowsMACAddresses() ([]string, error) {
	cmd := exec.Command("wmic", "path", "win32_networkadapter", "where", "physicaladapter=true", "get", "MacAddress", "/value")
	output, err := cmd.Output()
	switch err {
	case nil:
		return parseWmicMultipleValues(string(output), "MacAddress="), nil
	default:
		return nil, err
	}
}

// getWindowsDiskSerials retrieves disk serial numbers using wmic
func getWindowsDiskSerials() ([]string, error) {
	cmd := exec.Command("wmic", "diskdrive", "get", "SerialNumber", "/value")
	output, err := cmd.Output()
	switch err {
	case nil:
		return parseWmicMultipleValues(string(output), "SerialNumber="), nil
	default:
		return nil, err
	}
}
