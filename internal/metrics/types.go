package metrics

import "time"

// ZramStats contains the aggregated statistics for zram usage
type ZramStats struct {
	// System Info (captured once)
	SystemInfo struct {
		TotalMemory   uint64    `json:"total_memory"`
		KernelVersion string    `json:"kernel_version"`
		StartTime     time.Time `json:"start_time"`
	} `json:"system_info"`

	// Configuration (captured once)
	Config struct {
		Algorithm   string `json:"algorithm"`
		InitialSize uint64 `json:"initial_size"`
	} `json:"config"`

	CompressionStats struct {
		BestRatio      float64 `json:"best_ratio"`
		WorstRatio     float64 `json:"worst_ratio"`
		TotalRatio     float64 `json:"total_ratio"` // For average calculation
		SampleCount    uint64  `json:"sample_count"`
		ExcellentCount uint64  `json:"excellent_count"` // minutes with ratio <= 0.2
		GoodCount      uint64  `json:"good_count"`      // minutes with 0.2 < ratio <= 0.3
		FairCount      uint64  `json:"fair_count"`      // minutes with 0.3 < ratio <= 0.4
		PoorCount      uint64  `json:"poor_count"`      // minutes with ratio > 0.4
	} `json:"compression_stats"`

	MemoryStats struct {
		PeakUsage     uint64 `json:"peak_usage"`
		MinUsage      uint64 `json:"min_usage"`
		TotalUsage    uint64 `json:"total_usage"`    // For average calculation
		LowCount      uint64 `json:"low_count"`      // minutes at 0-25%
		MediumCount   uint64 `json:"medium_count"`   // minutes at 25-50%
		HighCount     uint64 `json:"high_count"`     // minutes at 50-75%
		CriticalCount uint64 `json:"critical_count"` // minutes at >75%
	} `json:"memory_stats"`

	SystemImpact struct {
		OOMEvents        uint64 `json:"oom_events"`
		MaxSwapUsed      uint64 `json:"max_swap_used"`
		SwapPressureTime uint64 `json:"swap_pressure_time"` // minutes under pressure
	} `json:"system_impact"`

	TimeAnalysis struct {
		HourlyUsage   [24]uint64 `json:"hourly_usage"`   // Accumulated usage per hour
		HourlySamples [24]uint64 `json:"hourly_samples"` // Sample count per hour
	} `json:"time_analysis"`
}

// UpdateStats updates the running statistics with new measurements
func (s *ZramStats) UpdateStats(origSize, compSize, memUsed uint64) {
	// Calculate compression ratio
	ratio := float64(compSize) / float64(origSize)
	if origSize == 0 {
		ratio = 1.0 // Avoid division by zero
	}

	// Update compression stats
	if s.CompressionStats.SampleCount == 0 || ratio < s.CompressionStats.BestRatio {
		s.CompressionStats.BestRatio = ratio
	}
	if ratio > s.CompressionStats.WorstRatio {
		s.CompressionStats.WorstRatio = ratio
	}
	s.CompressionStats.TotalRatio += ratio
	s.CompressionStats.SampleCount++

	// Update compression distribution
	switch {
	case ratio <= 0.2:
		s.CompressionStats.ExcellentCount++
	case ratio <= 0.3:
		s.CompressionStats.GoodCount++
	case ratio <= 0.4:
		s.CompressionStats.FairCount++
	default:
		s.CompressionStats.PoorCount++
	}

	// Update memory stats
	if s.MemoryStats.MinUsage == 0 || memUsed < s.MemoryStats.MinUsage {
		s.MemoryStats.MinUsage = memUsed
	}
	if memUsed > s.MemoryStats.PeakUsage {
		s.MemoryStats.PeakUsage = memUsed
	}
	s.MemoryStats.TotalUsage += memUsed

	// Update memory distribution
	usagePercent := float64(memUsed) / float64(s.Config.InitialSize)
	switch {
	case usagePercent <= 0.25:
		s.MemoryStats.LowCount++
	case usagePercent <= 0.50:
		s.MemoryStats.MediumCount++
	case usagePercent <= 0.75:
		s.MemoryStats.HighCount++
	default:
		s.MemoryStats.CriticalCount++
	}

	// Update hourly stats
	hour := time.Now().Hour()
	s.TimeAnalysis.HourlyUsage[hour] += memUsed
	s.TimeAnalysis.HourlySamples[hour]++
}
