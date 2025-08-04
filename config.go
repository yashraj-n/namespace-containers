package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// ContainerConfig holds configuration for the container
type ContainerConfig struct {
	Hostname     string
	RootFS       string
	Mounts       []Mount
	NetworkCIDR  string
	HostIP       string
	ContainerIP  string
	Command      []string
}

// Mount represents a bind mount from host to container
type Mount struct {
	Source      string // Host path
	Destination string // Container path
	ReadOnly    bool
}

// NewDefaultConfig returns a configuration with sensible defaults
func NewDefaultConfig() *ContainerConfig {
	return &ContainerConfig{
		Hostname:     "container",
		RootFS:       "./namespace_fs",
		NetworkCIDR:  "192.168.1.0/24",
		HostIP:       "192.168.1.1",
		ContainerIP:  "192.168.1.2",
		Mounts:       []Mount{},
	}
}

// ParseFlags parses command line flags and returns a configured ContainerConfig
func ParseFlags(args []string) (*ContainerConfig, error) {
	config := NewDefaultConfig()
	
	flagSet := flag.NewFlagSet("container", flag.ExitOnError)
	
	hostname := flagSet.String("hostname", config.Hostname, "Container hostname")
	rootfs := flagSet.String("rootfs", config.RootFS, "Container root filesystem path")
	networkCIDR := flagSet.String("network", config.NetworkCIDR, "Container network CIDR")
	hostIP := flagSet.String("host-ip", config.HostIP, "Host IP address")
	containerIP := flagSet.String("container-ip", config.ContainerIP, "Container IP address")
	
	var mountFlags multiString
	flagSet.Var(&mountFlags, "mount", "Bind mount (format: host_path:container_path[:ro]). Can be specified multiple times")
	
	// Parse everything except the command
	var commandStart int
	for i, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			commandStart = i
			break
		}
	}
	
	if commandStart == 0 {
		config.Command = args
	} else {
		err := flagSet.Parse(args[:commandStart])
		if err != nil {
			return nil, fmt.Errorf("failed to parse flags: %v", err)
		}
		config.Command = args[commandStart:]
	}
	
	// Update config with parsed values
	config.Hostname = *hostname
	config.RootFS = *rootfs
	config.NetworkCIDR = *networkCIDR
	config.HostIP = *hostIP
	config.ContainerIP = *containerIP
	
	// Parse mount options
	for _, mountStr := range mountFlags {
		mount, err := parseMount(mountStr)
		if err != nil {
			return nil, fmt.Errorf("invalid mount specification '%s': %v", mountStr, err)
		}
		config.Mounts = append(config.Mounts, mount)
	}
	
	// Add default /app mount if none specified
	if len(config.Mounts) == 0 {
		cwd, err := os.Getwd()
		if err == nil {
			config.Mounts = append(config.Mounts, Mount{
				Source:      cwd,
				Destination: "/app",
				ReadOnly:    false,
			})
		}
	}
	
	if len(config.Command) == 0 {
		return nil, fmt.Errorf("no command specified")
	}
	
	return config, nil
}

// parseMount parses a mount string in the format "host_path:container_path[:ro]"
func parseMount(mountStr string) (Mount, error) {
	parts := strings.Split(mountStr, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return Mount{}, fmt.Errorf("mount format should be host_path:container_path[:ro]")
	}
	
	mount := Mount{
		Source:      parts[0],
		Destination: parts[1],
		ReadOnly:    false,
	}
	
	if len(parts) == 3 && parts[2] == "ro" {
		mount.ReadOnly = true
	}
	
	return mount, nil
}

// multiString allows multiple values for the same flag
type multiString []string

func (m *multiString) String() string {
	return strings.Join(*m, ",")
}

func (m *multiString) Set(value string) error {
	*m = append(*m, value)
	return nil
}