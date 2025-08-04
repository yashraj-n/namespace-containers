//go:build linux
// +build linux

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
)

// SetupFilesystem prepares the container's filesystem including mounts
func SetupFilesystem(config *ContainerConfig) error {
	// Setup bind mounts BEFORE chroot so source paths are still accessible
	if err := setupBindMounts(config); err != nil {
		return fmt.Errorf("failed to setup bind mounts: %v", err)
	}

	// Set hostname
	if err := syscall.Sethostname([]byte(config.Hostname)); err != nil {
		return fmt.Errorf("failed to set hostname: %v", err)
	}

	// Change root to the container filesystem
	if err := syscall.Chroot(config.RootFS); err != nil {
		return fmt.Errorf("failed to chroot to %s: %v", config.RootFS, err)
	}

	// Change working directory to root
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("failed to change directory to /: %v", err)
	}

	// Copy resolv.conf for DNS resolution
	if err := copyResolvConf(); err != nil {
		return fmt.Errorf("failed to setup DNS: %v", err)
	}

	// Mount proc filesystem
	if err := syscall.Mount("proc", "proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("failed to mount proc: %v", err)
	}

	// Mount /dev/pts for pseudo-terminals
	if err := syscall.Mount("devpts", "/dev/pts", "devpts", 0, ""); err != nil {
		return fmt.Errorf("failed to mount devpts: %v", err)
	}

	return nil
}

// CleanupFilesystem unmounts filesystems
func CleanupFilesystem() error {
	var lastErr error

	// Unmount in reverse order
	if err := syscall.Unmount("/dev/pts", 0); err != nil {
		lastErr = err
	}

	if err := syscall.Unmount("proc", 0); err != nil {
		lastErr = err
	}

	return lastErr
}

// setupBindMounts creates bind mounts for the container
func setupBindMounts(config *ContainerConfig) error {
	for _, mount := range config.Mounts {
		if err := createBindMount(mount, config.RootFS); err != nil {
			return fmt.Errorf("failed to create bind mount %s -> %s: %v",
				mount.Source, mount.Destination, err)
		}
	}
	return nil
}

// createBindMount creates a single bind mount
func createBindMount(mount Mount, rootFS string) error {
	// Verify source exists
	if _, err := os.Stat(mount.Source); os.IsNotExist(err) {
		return fmt.Errorf("source path does not exist: %s", mount.Source)
	}

	// Create the mount point in the rootFS
	mountPoint := filepath.Join(rootFS, mount.Destination)
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point %s: %v", mountPoint, err)
	}

	// Determine mount flags
	flags := uintptr(syscall.MS_BIND)
	if mount.ReadOnly {
		flags |= syscall.MS_RDONLY
	}

	// Create the bind mount
	if err := syscall.Mount(mount.Source, mountPoint, "", flags, ""); err != nil {
		return fmt.Errorf("failed to bind mount: %v", err)
	}

	// If read-only, remount with MS_RDONLY (required for bind mounts)
	if mount.ReadOnly {
		remountFlags := flags | syscall.MS_REMOUNT
		if err := syscall.Mount("", mountPoint, "", remountFlags, ""); err != nil {
			return fmt.Errorf("failed to remount as read-only: %v", err)
		}
	}

	return nil
}

// copyResolvConf sets up DNS resolution in the container, in this case 8.8.8.8 google's dns
func copyResolvConf() error {
	if err := os.MkdirAll("/etc", 0755); err != nil {
		return fmt.Errorf("failed to create /etc directory: %v", err)
	}

	content := []byte("nameserver 8.8.8.8\nnameserver 8.8.4.4\n")
	if err := ioutil.WriteFile("/etc/resolv.conf", content, 0644); err != nil {
		return fmt.Errorf("failed to write resolv.conf: %v", err)
	}

	return nil
}

// PrepareRootFS ensures the root filesystem directory exists and is properly set up
func PrepareRootFS(rootfsPath string) error {
	// Check if rootfs exists
	if _, err := os.Stat(rootfsPath); os.IsNotExist(err) {
		return fmt.Errorf("root filesystem path %s does not exist", rootfsPath)
	}

	// Ensure required directories exist in the rootfs
	requiredDirs := []string{
		filepath.Join(rootfsPath, "proc"), // Required for proc filesystem, else ps command will not work
		filepath.Join(rootfsPath, "dev"),
		filepath.Join(rootfsPath, "dev/pts"), // Required for pseudo-terminals, else ls -l /dev/pts will not work
		filepath.Join(rootfsPath, "etc"),     // Required for DNS resolution, else ping google.com will not work
		filepath.Join(rootfsPath, "app"),
		filepath.Join(rootfsPath, "tmp"),
	}

	for _, dir := range requiredDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	return nil
}
