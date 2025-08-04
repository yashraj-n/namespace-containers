//go:build linux
// +build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// RunContainer starts a new container with the given configuration
func RunContainer(config *ContainerConfig) error {
	// Validate configuration
	if err := validateConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}

	// Prepare the root filesystem
	if err := PrepareRootFS(config.RootFS); err != nil {
		return fmt.Errorf("failed to prepare root filesystem: %v", err)
	}

	logInfo("Starting container with command: %v", config.Command)
	if len(config.Mounts) > 0 {
		logInfo("Mounts configured: %d", len(config.Mounts))
	}

	// Create child process with new namespaces
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, config.Command...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment variables for the child process
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("CONTAINER_HOSTNAME=%s", config.Hostname),
		fmt.Sprintf("CONTAINER_ROOTFS=%s", config.RootFS),
		fmt.Sprintf("CONTAINER_NETWORK_CIDR=%s", config.NetworkCIDR),
		fmt.Sprintf("CONTAINER_HOST_IP=%s", config.HostIP),
		fmt.Sprintf("CONTAINER_CONTAINER_IP=%s", config.ContainerIP),
	)

	// Add mount information to environment
	for i, mount := range config.Mounts {
		readonly := "false"
		if mount.ReadOnly {
			readonly = "true"
		}
		cmd.Env = append(cmd.Env,
			fmt.Sprintf("CONTAINER_MOUNT_%d_SOURCE=%s", i, mount.Source),
			fmt.Sprintf("CONTAINER_MOUNT_%d_DEST=%s", i, mount.Destination),
			fmt.Sprintf("CONTAINER_MOUNT_%d_READONLY=%s", i, readonly),
		)
	}
	cmd.Env = append(cmd.Env, fmt.Sprintf("CONTAINER_MOUNT_COUNT=%d", len(config.Mounts)))

	// Configure namespaces for the child process
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | // UTS namespace (hostname)
			syscall.CLONE_NEWPID | // PID namespace
			syscall.CLONE_NEWNS | // Mount namespace
			syscall.CLONE_NEWNET, // Network namespace
		Unshareflags: syscall.CLONE_NEWNS, // Unshare mount namespace
	}

	// Start the container process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}

	logInfo("Container started with PID: %d \n\n", cmd.Process.Pid)

	// Setup networking for the container
	if err := SetupNetworking(cmd.Process.Pid, config); err != nil {
		// Kill the container process if networking setup fails
		cmd.Process.Kill()
		return fmt.Errorf("failed to setup networking: %v", err)
	}

	logInfo("Network setup completed")

	// Wait for the container to finish
	err := cmd.Wait()

	// Clean up network rules
	CleanupNetwork(config)
	logInfo("Container finished")

	return err
}

// RunChildProcess runs inside the container namespace
func RunChildProcess(command []string) error {
	// Parse configuration from environment variables
	config, err := configFromEnv()
	if err != nil {
		return fmt.Errorf("failed to parse container config from environment: %v", err)
	}

	logDebug("Child process starting with config: hostname=%s, rootfs=%s",
		config.Hostname, config.RootFS)

	// Setup cgroups
	cgroupConfig := NewDefaultCgroupConfig()
	if err := SetupCgroups(cgroupConfig); err != nil {
		return fmt.Errorf("failed to setup cgroups: %v", err)
	}

	// Setup filesystem (including mounts)
	if err := SetupFilesystem(config); err != nil {
		return fmt.Errorf("failed to setup filesystem: %v", err)
	}

	logDebug("Filesystem setup completed")

	// Prepare and execute the user command
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set working directory to /app if it exists
	if dirExists("/app") {
		cmd.Dir = "/app"
		logDebug("Working directory set to /app")
	}

	// Run the command
	err = cmd.Run()

	// Cleanup filesystem
	if cleanupErr := CleanupFilesystem(); cleanupErr != nil {
		logError("Failed to cleanup filesystem: %v", cleanupErr)
	}

	return err
}

// validateConfig validates the container configuration
func validateConfig(config *ContainerConfig) error {
	if err := validateCommand(config.Command); err != nil {
		return err
	}

	if err := validatePath(config.RootFS); err != nil {
		return fmt.Errorf("invalid root filesystem path: %v", err)
	}

	for i, mount := range config.Mounts {
		if err := validatePath(mount.Source); err != nil {
			return fmt.Errorf("invalid mount source path for mount %d: %v", i, err)
		}

		if mount.Destination == "" {
			return fmt.Errorf("mount destination cannot be empty for mount %d", i)
		}
	}

	return nil
}

// configFromEnv reconstructs container configuration from environment variables
func configFromEnv() (*ContainerConfig, error) {
	config := &ContainerConfig{
		Hostname:    os.Getenv("CONTAINER_HOSTNAME"),
		RootFS:      os.Getenv("CONTAINER_ROOTFS"),
		NetworkCIDR: os.Getenv("CONTAINER_NETWORK_CIDR"),
		HostIP:      os.Getenv("CONTAINER_HOST_IP"),
		ContainerIP: os.Getenv("CONTAINER_CONTAINER_IP"),
	}

	// Parse mount information
	mountCountStr := os.Getenv("CONTAINER_MOUNT_COUNT")
	if mountCountStr != "" {
		mountCount, err := parsePID(mountCountStr) // reusing parsePID for int parsing
		if err != nil {
			return nil, fmt.Errorf("invalid mount count: %v", err)
		}

		for i := 0; i < mountCount; i++ {
			source := os.Getenv(fmt.Sprintf("CONTAINER_MOUNT_%d_SOURCE", i))
			dest := os.Getenv(fmt.Sprintf("CONTAINER_MOUNT_%d_DEST", i))
			readonlyStr := os.Getenv(fmt.Sprintf("CONTAINER_MOUNT_%d_READONLY", i))

			mount := Mount{
				Source:      source,
				Destination: dest,
				ReadOnly:    readonlyStr == "true",
			}

			config.Mounts = append(config.Mounts, mount)
		}
	}

	return config, nil
}
