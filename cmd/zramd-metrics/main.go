package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	"zramd/internal/metrics"
	"zramd/internal/zram"
)

var (
	Version    = "0.0.0"
	CommitDate = "?"
)

func main() {
	deviceID := flag.Int("device", 0, "zram device ID to monitor")
	interval := flag.Duration("interval", time.Minute, "collection interval")
	flag.Parse()

	// Check for root privileges
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "error: root privileges are required")
		os.Exit(1)
	}

	// Check if zram module is loaded
	if !zram.IsLoaded() {
		fmt.Fprintln(os.Stderr, "error: zram module is not loaded")
		os.Exit(1)
	}

	// Check if specified device exists
	if !metrics.DeviceExists(*deviceID) {
		fmt.Fprintf(os.Stderr, "error: zram%d device does not exist\n", *deviceID)
		os.Exit(1)
	}

	// Get current device configuration
	algorithm, err := metrics.GetDeviceAlgorithm(*deviceID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: getting device algorithm: %v\n", err)
		os.Exit(1)
	}

	initialSize, err := metrics.GetDeviceSize(*deviceID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: getting device size: %v\n", err)
		os.Exit(1)
	}

	// Initialize or load stats
	stats, err := metrics.InitializeStats(algorithm, initialSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: initializing stats: %v\n", err)
		os.Exit(1)
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create ticker for regular collection
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	fmt.Printf("Starting metrics collection for zram%d every %v\n", *deviceID, *interval)

	for {
		select {
		case <-ticker.C:
			if err := collectAndStore(*deviceID, stats); err != nil {
				fmt.Fprintf(os.Stderr, "error: collecting metrics: %v\n", err)
			}

		case sig := <-sigChan:
			fmt.Printf("Received signal %v, saving stats and shutting down\n", sig)
			if err := metrics.SaveStats(stats); err != nil {
				fmt.Fprintf(os.Stderr, "error: saving final stats: %v\n", err)
			}
			return
		}
	}
}

func collectAndStore(deviceID int, stats *metrics.ZramStats) error {
	// Collect metrics
	origSize, compSize, memUsed, err := metrics.CollectMetrics(deviceID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: collecting metrics: %v\n", err)
		return err
	}

	// Debug logging
	fmt.Printf("Collected metrics - Original: %.2f GB, Compressed: %.2f GB, Memory Used: %.2f GB\n", float64(origSize)/1024/1024/1024, float64(compSize)/1024/1024/1024, float64(memUsed)/1024/1024/1024)

	// Update running statistics
	stats.UpdateStats(origSize, compSize, memUsed)

	// Save to disk periodically
	if err := metrics.SaveStats(stats); err != nil {
		return fmt.Errorf("saving stats: %w", err)
	}

	return nil
}
