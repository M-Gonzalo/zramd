package metrics

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	sysfsPath = "/sys/block/zram%d"
)

// CollectMetrics gathers current metrics from a zram device
func CollectMetrics(deviceID int) (uint64, uint64, uint64, error) {
	// Try reading from mm_stat first (newer kernels)
	stats, err := readMMStats(deviceID)
	if err == nil {
		// mm_stat format: orig_data_size compr_data_size mem_used_total ...
		return stats[0], stats[1], stats[2], nil
	}

	// Fall back to individual files (older kernels)
	origSize, err := ReadSysfsValue(deviceID, "orig_data_size")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("reading original size: %w", err)
	}

	comprSize, err := ReadSysfsValue(deviceID, "compr_data_size")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("reading compressed size: %w", err)
	}

	memUsed, err := ReadSysfsValue(deviceID, "mem_used_total")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("reading memory used: %w", err)
	}

	return origSize, comprSize, memUsed, nil
}

// readMMStats reads and parses the mm_stat file for a zram device
func readMMStats(deviceID int) ([]uint64, error) {
	path := fmt.Sprintf(sysfsPath+"/%s", deviceID, "mm_stat")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Split the stats and convert to uint64
	fields := strings.Fields(string(data))
	stats := make([]uint64, len(fields))
	for i, field := range fields[:3] { // We only need the first 3 values
		stats[i], err = strconv.ParseUint(field, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing mm_stat field %d: %w", i, err)
		}
	}

	return stats, nil
}

// ReadSysfsValue reads a uint64 value from a zram sysfs file
func ReadSysfsValue(deviceID int, metric string) (uint64, error) {
	path := fmt.Sprintf(sysfsPath+"/%s", deviceID, metric)
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	// Trim whitespace and convert to uint64
	value, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing %s: %w", metric, err)
	}

	return value, nil
}

// DeviceExists checks if a zram device exists and is accessible
func DeviceExists(deviceID int) bool {
	path := fmt.Sprintf(sysfsPath, deviceID)
	_, err := os.Stat(path)
	return err == nil
}

// GetDeviceAlgorithm reads the current compression algorithm for a device
func GetDeviceAlgorithm(deviceID int) (string, error) {
	path := fmt.Sprintf(sysfsPath+"/%s", deviceID, "comp_algorithm")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// The format is usually "[algo] algo algo", we want the selected one
	algorithms := strings.Fields(string(data))
	for _, algo := range algorithms {
		if strings.HasPrefix(algo, "[") && strings.HasSuffix(algo, "]") {
			return strings.Trim(algo, "[]"), nil
		}
	}

	// If no algorithm is marked as selected, return the first one
	if len(algorithms) > 0 {
		return algorithms[0], nil
	}

	return "", fmt.Errorf("no compression algorithm found")
}

// GetDeviceSize reads the current size of the device
func GetDeviceSize(deviceID int) (uint64, error) {
	return ReadSysfsValue(deviceID, "disksize")
}
