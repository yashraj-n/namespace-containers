//go:build linux
// +build linux

package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// must panics if err is not nil
func must(err error) {
	if err != nil {
		log.Fatalf("error: %v", err)
	}
}

// mustf panics with a formatted message if err is not nil
func mustf(err error, format string, args ...interface{}) {
	if err != nil {
		log.Fatalf(format+": %v", append(args, err)...)
	}
}

// checkRoot verifies that the program is running as root
func checkRoot() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("this program must be run as root")
	}
	return nil
}

// parsePID converts a string to a PID
func parsePID(pidStr string) (int, error) {
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PID: %s", pidStr)
	}
	if pid <= 0 {
		return 0, fmt.Errorf("PID must be positive: %d", pid)
	}
	return pid, nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return !os.IsNotExist(err) && info.IsDir()
}

// ensureDir creates a directory if it doesn't exist
func ensureDir(path string, perm os.FileMode) error {
	if !dirExists(path) {
		return os.MkdirAll(path, perm)
	}
	return nil
}

// validatePath checks if a path is valid and accessible
func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	if !fileExists(path) {
		return fmt.Errorf("path does not exist: %s", path)
	}

	return nil
}

// validateCommand checks if a command is valid
func validateCommand(command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("command cannot be empty")
	}

	if command[0] == "" {
		return fmt.Errorf("command name cannot be empty")
	}

	return nil
}

// printConfig prints the container configuration in a readable format
func printConfig(config *ContainerConfig) {
	fmt.Printf("Container Configuration:\n")
	fmt.Printf("  Hostname: %s\n", config.Hostname)
	fmt.Printf("  Root FS: %s\n", config.RootFS)
	fmt.Printf("  Network: %s\n", config.NetworkCIDR)
	fmt.Printf("  Host IP: %s\n", config.HostIP)
	fmt.Printf("  Container IP: %s\n", config.ContainerIP)
	fmt.Printf("  Command: %s\n", strings.Join(config.Command, " "))

	if len(config.Mounts) > 0 {
		fmt.Printf("  Mounts:\n")
		for _, mount := range config.Mounts {
			readOnly := ""
			if mount.ReadOnly {
				readOnly = " (read-only)"
			}
			fmt.Printf("    %s -> %s%s\n", mount.Source, mount.Destination, readOnly)
		}
	}
}

// logDebug prints debug information if debug mode is enabled
func logDebug(format string, args ...interface{}) {
	if os.Getenv("DEBUG") != "" {
		log.Printf("[DEBUG] "+format, args...)
	}
}

// logInfo prints informational messages
func logInfo(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

// logError prints error messages
func logError(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}

// contains checks if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// unique removes duplicate strings from a slice
func unique(slice []string) []string {
	keys := make(map[string]bool)
	result := []string{}

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}
