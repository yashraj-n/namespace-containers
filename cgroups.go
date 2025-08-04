//go:build linux
// +build linux

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

const (
	cgroupBasePath = "/sys/fs/cgroup"
	cgroupName     = "namespace_test"
)

// CgroupConfig holds cgroup configuration options
type CgroupConfig struct {
	Name        string
	MaxPids     string
	MemoryLimit string
	CPUQuota    string
	CPUPeriod   string
}

// NewDefaultCgroupConfig returns a cgroup config with sensible defaults
func NewDefaultCgroupConfig() *CgroupConfig {
	return &CgroupConfig{
		Name:        cgroupName,
		MaxPids:     "max",    // No limit by default
		MemoryLimit: "",       // No limit by default
		CPUQuota:    "",       // No limit by default
		CPUPeriod:   "100000", // Default period (100ms)
	}
}

// SetupCgroups creates and configures cgroups for the container
func SetupCgroups(config *CgroupConfig) error {
	cgroupPath := filepath.Join(cgroupBasePath, config.Name)

	// Create cgroup directory if it doesn't exist
	if _, err := os.Stat(cgroupPath); os.IsNotExist(err) {
		if err := os.MkdirAll(cgroupPath, 0755); err != nil {
			return fmt.Errorf("failed to create cgroup directory %s: %v", cgroupPath, err)
		}
	}

	// Set process limits
	if err := setCgroupValue(cgroupPath, "pids.max", config.MaxPids); err != nil {
		return fmt.Errorf("failed to set pids.max: %v", err)
	}

	// Set memory limits if specified
	if config.MemoryLimit != "" {
		if err := setCgroupValue(cgroupPath, "memory.max", config.MemoryLimit); err != nil {
			return fmt.Errorf("failed to set memory.max: %v", err)
		}
	}

	// Set CPU limits if specified
	if config.CPUQuota != "" {
		if err := setCgroupValue(cgroupPath, "cpu.max",
			fmt.Sprintf("%s %s", config.CPUQuota, config.CPUPeriod)); err != nil {
			return fmt.Errorf("failed to set cpu.max: %v", err)
		}
	}

	// Add current process to the cgroup
	pid := os.Getpid()
	if err := setCgroupValue(cgroupPath, "cgroup.procs", strconv.Itoa(pid)); err != nil {
		return fmt.Errorf("failed to add process to cgroup: %v", err)
	}

	return nil
}

// CleanupCgroups removes the cgroup (optional, as it will be cleaned up automatically)
func CleanupCgroups(config *CgroupConfig) error {
	cgroupPath := filepath.Join(cgroupBasePath, config.Name)

	// Remove the cgroup directory
	if err := os.RemoveAll(cgroupPath); err != nil {
		return fmt.Errorf("failed to remove cgroup directory: %v", err)
	}

	return nil
}

// setCgroupValue writes a value to a cgroup file
func setCgroupValue(cgroupPath, filename, value string) error {
	filePath := filepath.Join(cgroupPath, filename)

	// Check if the file exists (some cgroup features might not be available)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File doesn't exist, skip this setting
		return nil
	}

	if err := ioutil.WriteFile(filePath, []byte(value), 0644); err != nil {
		return fmt.Errorf("failed to write to %s: %v", filePath, err)
	}

	return nil
}

// GetCgroupStats reads current cgroup statistics
func GetCgroupStats(config *CgroupConfig) (map[string]string, error) {
	cgroupPath := filepath.Join(cgroupBasePath, config.Name)
	stats := make(map[string]string)

	// List of files to read for statistics
	statFiles := []string{
		"pids.current",
		"memory.current",
		"memory.max",
		"cpu.stat",
	}

	for _, file := range statFiles {
		filePath := filepath.Join(cgroupPath, file)
		if data, err := ioutil.ReadFile(filePath); err == nil {
			stats[file] = string(data)
		}
	}

	return stats, nil
}

// ParseMemorySize converts human-readable memory sizes to bytes
func ParseMemorySize(size string) (string, error) {
	if size == "" {
		return "", nil
	}

	// Handle common suffixes
	switch size[len(size)-1:] {
	case "K", "k":
		return size[:len(size)-1] + "000", nil
	case "M", "m":
		return size[:len(size)-1] + "000000", nil
	case "G", "g":
		return size[:len(size)-1] + "000000000", nil
	default:
		// Assume it's already in bytes
		return size, nil
	}
}
