//go:build linux
// +build linux

package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "help", "--help", "-h":
		printUsage()
		return
	case "version", "--version", "-v":
		printVersion()
		return
	}

	// Check if running as root for commands that need it
	if err := checkRoot(); err != nil {
		logError("%v", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		handleRun(os.Args[2:])
	case "child":
		handleChild(os.Args[2:])
	default:
		logError("Unknown command: %s", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func handleRun(args []string) {
	config, err := ParseFlags(args)
	if err != nil {
		logError("Configuration error: %v", err)
		os.Exit(1)
	}

	// Print configuration if debug mode is enabled
	if os.Getenv("DEBUG") != "" {
		printConfig(config)
	}

	// Run the container
	if err := RunContainer(config); err != nil {
		logError("Container failed: %v", err)
		os.Exit(1)
	}
}

func handleChild(args []string) {
	if len(args) == 0 {
		logError("No command specified for child process")
		os.Exit(1)
	}

	if err := RunChildProcess(args); err != nil {
		logError("Child process failed: %v", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`Container Runtime - A simple Linux container implementation

Usage: %s [OPTIONS] COMMAND [ARG...]

Commands:
  run    Run a command in a new container
  help   Show this help message

Options for 'run' command:
  --hostname HOSTNAME        Set container hostname (default: container)
  --rootfs PATH             Path to container root filesystem (default: ./namespace_fs)
  --network CIDR            Container network CIDR (default: 192.168.1.0/24)
  --host-ip IP              Host IP address (default: 192.168.1.1)
  --container-ip IP         Container IP address (default: 192.168.1.2)
  --mount HOST:CONTAINER[:ro] Bind mount host directory to container
                            Can be specified multiple times
                            Add :ro for read-only mounts

Examples:
  # Run bash in a container with current directory mounted to /app
  sudo %s run /bin/bash

  # Run with custom mounts
  sudo %s run --mount /home/user/code:/app --mount /tmp:/tmp:ro /bin/bash

  # Run with custom hostname and network
  sudo %s run --hostname mycontainer --network 10.0.0.0/24 /bin/sh

Environment Variables:
  DEBUG=1                   Enable debug output

Notes:
  - This program must be run as root
  - The rootfs directory must exist and contain a basic Linux filesystem
  - iptables is required for network functionality
  - The container will have network access through NAT

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func printVersion() {
	fmt.Println("Container Runtime v1.0.0")
	fmt.Println("A simple Linux container implementation using namespaces by Yashraj Narke")
}
