package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	metricsDir      = "/var/log/zramd/metrics"
	metricsFile     = "zram_stats.json"
	metricsFileMode = 0644
	metricsDirMode  = 0755
	backupFileExt   = ".bak"
)

// InitializeStats creates a new stats file if it doesn't exist
func InitializeStats(algorithm string, initialSize uint64) (*ZramStats, error) {
	// Ensure metrics directory exists
	if err := os.MkdirAll(metricsDir, metricsDirMode); err != nil {
		return nil, fmt.Errorf("creating metrics directory: %w", err)
	}

	// Read total memory from /proc/meminfo
	memTotal, err := getMemTotal()
	if err != nil {
		return nil, fmt.Errorf("reading total memory: %w", err)
	}

	// Create initial stats structure
	stats := &ZramStats{}
	stats.SystemInfo.TotalMemory = memTotal
	stats.SystemInfo.KernelVersion = getKernelVersion()
	stats.SystemInfo.StartTime = time.Now()
	stats.Config.Algorithm = algorithm
	stats.Config.InitialSize = initialSize

	// Write initial stats file if it doesn't exist
	if _, err := os.Stat(getStatsPath()); os.IsNotExist(err) {
		if err := writeStats(stats); err != nil {
			return nil, fmt.Errorf("writing initial stats: %w", err)
		}
	}

	return stats, nil
}

// LoadStats reads the current stats from file
func LoadStats() (*ZramStats, error) {
	data, err := os.ReadFile(getStatsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("stats file not found, run initialize first")
		}
		return nil, err
	}

	var stats ZramStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, fmt.Errorf("parsing stats file: %w", err)
	}

	return &stats, nil
}

// SaveStats writes the current stats to file with backup
func SaveStats(stats *ZramStats) error {
	// First write to backup file
	backupPath := getStatsPath() + backupFileExt
	if err := writeStatsToPath(stats, backupPath); err != nil {
		return fmt.Errorf("writing backup: %w", err)
	}

	// Then write to main file
	if err := writeStats(stats); err != nil {
		return fmt.Errorf("writing stats: %w", err)
	}

	return nil
}

func getStatsPath() string {
	return filepath.Join(metricsDir, metricsFile)
}

func writeStats(stats *ZramStats) error {
	return writeStatsToPath(stats, getStatsPath())
}

func writeStatsToPath(stats *ZramStats, path string) error {
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding stats: %w", err)
	}

	if err := os.WriteFile(path, data, metricsFileMode); err != nil {
		return fmt.Errorf("writing stats file: %w", err)
	}

	return nil
}

func getMemTotal() (uint64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				return 0, fmt.Errorf("invalid MemTotal line: %s", line)
			}
			kb, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("parsing MemTotal: %w", err)
			}
			return kb * 1024, nil // Convert KB to bytes
		}
	}
	return 0, fmt.Errorf("MemTotal not found in /proc/meminfo")
}

func getKernelVersion() string {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(data))
}
