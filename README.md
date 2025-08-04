<h1 align="center">
    <img src="https://i.ibb.co/5xBRh6xd/github-header-banner.png" alt="Namespace Containers">
</h1>

<h3 align="center">
    🐧 A lightweight Linux container runtime implementation using namespaces
</h3>

<p align="center">
    <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go" alt="Go Version"></a>
    <a href="https://github.com/yashraj-n/namespace-containers/blob/master/LICENSE"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License"></a>
    <a href="https://github.com/yashraj-n/namespace-containers/releases"><img src="https://img.shields.io/github/v/release/yashraj-n/namespace-containers?include_prereleases" alt="Release"></a>
    <a href="https://goreportcard.com/report/github.com/yashraj-n/namespace-containers"><img src="https://goreportcard.com/badge/github.com/yashraj-n/namespace-containers" alt="Go Report Card"></a>
</p>

<p align="center">
    <strong>
        <a href="#overview">Overview</a>
        •
        <a href="#features">Features</a>
        •
        <a href="#installation">Installation</a>
        •
        <a href="#usage">Usage</a>
        •
        <a href="#examples">Examples</a>
        •
        <a href="#development">Development</a>
        •
        <a href="#contributing">Contributing</a>
    </strong>
</p>

---

## Overview

**Namespace Containers** is a lightweight, educational container runtime implementation written in Go that demonstrates the core concepts behind Linux containers. Built from scratch using Linux namespaces, cgroups, and network virtualization, this project provides a minimal yet functional container environment.

## Features

### Core Container Features
- **Process Isolation**: Uses PID namespaces to isolate processes
- **Filesystem Isolation**: Mount namespaces with chroot for filesystem isolation
- **Network Isolation**: Dedicated network namespace with virtual ethernet pairs
- **Hostname Isolation**: UTS namespaces for independent hostname/domain
- **Resource Limits**: cgroups integration for CPU, memory, and process limits

### Networking
- **Virtual Network Interface**: Creates veth pairs for container networking
- **NAT Support**: Internet access through iptables NAT rules
- **Custom IP Configuration**: Configurable container and host IP addresses
- **DNS Resolution**: Automatic DNS setup with Google's public DNS

### Storage & Mounts
- **Bind Mounts**: Mount host directories into containers
- **Read-only Mounts**: Support for read-only bind mounts
- **Automatic /app Mount**: Current directory mounted to /app by default
- **Filesystem Preparation**: Automatic setup of required directories

### Resource Management
- **Memory Limits**: Set maximum memory usage
- **CPU Quotas**: Limit CPU usage with configurable periods
- **Process Limits**: Control maximum number of processes
- **Automatic Cleanup**: Resource cleanup on container exit

## Installation

### Prerequisites

- **Linux**: This project only works on Linux (uses Linux-specific syscalls)
- **Go 1.21+**: Required for building the project
- **Root Access**: Must be run as root for namespace operations
- **iptables**: Required for network functionality
- **Basic Linux Filesystem**: A root filesystem directory (see setup below)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/yashraj-n/namespace-containers.git
cd namespace-containers

# Build the binary
make build

# Or build manually
go build -o container main.go config.go filesystem.go network.go cgroups.go utils.go container.go
```

### Setting up Root Filesystem

You'll need a basic Linux filesystem to use as the container root:

On Debian based systems, you can use the following command to create a minimal root filesystem:
```bash
sudo apt install debootstrap -y
sudo debootstrap --variant=minbase focal ./namespace_fs http://archive.ubuntu.com/ubuntu
```
This will create a minimal root filesystem in the `namespace_fs` directory.
## Usage

### Basic Command Structure

```bash
sudo ./container [OPTIONS] COMMAND [ARG...]
```

### Available Commands

- `run`: Run a command in a new container
- `help`: Show help message
- `version`: Show version information

### Options for 'run' command

| Option | Description | Default |
|--------|-------------|---------|
| `--hostname HOSTNAME` | Set container hostname | `container` |
| `--rootfs PATH` | Path to container root filesystem | `./namespace_fs` |
| `--network CIDR` | Container network CIDR | `192.168.1.0/24` |
| `--host-ip IP` | Host IP address | `192.168.1.1` |
| `--container-ip IP` | Container IP address | `192.168.1.2` |
| `--mount HOST:CONTAINER[:ro]` | Bind mount (can specify multiple) | Current dir to `/app` |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `DEBUG=1` | Enable debug output |

## Examples

### Basic Usage

```bash
# Run bash in a container
sudo ./container run /bin/bash

# Run with custom hostname
sudo ./container run --hostname myapp /bin/sh

# Run a specific command
sudo ./container run --hostname webserver /usr/bin/python3 -m http.server
```

### Advanced Mounting

```bash
# Mount specific directories
sudo ./container run \
  --mount /home/user/code:/app \
  --mount /etc/passwd:/etc/passwd:ro \
  --mount /tmp:/tmp \
  /bin/bash

# Multiple read-only mounts
sudo ./container run \
  --mount /var/log:/logs:ro \
  --mount /etc:/host-etc:ro \
  --hostname logger \
  /bin/bash
```

### Custom Network Configuration

```bash
# Custom network setup
sudo ./container run \
  --network 10.0.0.0/24 \
  --host-ip 10.0.0.1 \
  --container-ip 10.0.0.2 \
  --hostname nettest \
  /bin/bash
```

### Debug Mode

```bash
# Enable debug output
DEBUG=1 sudo ./container run /bin/bash
```

## Architecture

```
┌─────────────────────────────────────────┐
│                Host System              │
│  ┌─────────────────────────────────────┐│
│  │            Container                ││
│  │  ┌─────────────────────────────────┐││
│  │  │        Your Application         │││
│  │  └─────────────────────────────────┘││
│  │                                     ││
│  │  Namespaces:                        ││
│  │  • PID (Process Isolation)          ││
│  │  • Mount (Filesystem Isolation)     ││
│  │  • Network (Network Isolation)      ││
│  │  • UTS (Hostname Isolation)         ││
│  └─────────────────────────────────────┘│
│                                         │
│  cgroups (Resource Limits)              │
│  iptables (Network NAT)                 │
│  veth pairs (Network Interface)         │
└─────────────────────────────────────────┘
```

## Development

### Project Structure

```
namespace-containers/
├── main.go          # Entry point and command handling
├── config.go        # Configuration parsing and management
├── container.go     # Core container lifecycle management
├── filesystem.go    # Filesystem setup and bind mounts
├── network.go       # Network configuration and veth setup
├── cgroups.go       # Resource limits and cgroups management
├── utils.go         # Utility functions and validation
├── Makefile         # Build configuration
└── README.md        # This file
```

### Development Setup

```bash
# Clone and setup
git clone https://github.com/yashraj-n/namespace-containers.git
cd namespace-containers

# Install dependencies
go mod download

# Build for development
go build -o container *.go


# Format code
go fmt ./...

# Vet code
go vet ./...
```

### Contributing

New features are welcome! Here's how to get started:

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/amazing-feature`)
3. **Make** your changes
4. **Test** your changes thoroughly
5. **Commit** your changes (`git commit -m 'Add amazing feature'`)
6. **Push** to the branch (`git push origin feature/amazing-feature`)
7. **Open** a Pull Request

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Add comments for exported functions
- Update documentation as needed

## Limitations

- **Linux Only**: Uses Linux-specific syscalls and features
- **Root Required**: Needs root privileges for namespace operations
- **Basic Implementation**: Educational focus, not production-ready
- **Limited Security**: Minimal security hardening
- **No Image Management**: Requires manual filesystem preparation

## Security Considerations

⚠️ **Warning**: This is an educational project and should not be used in production environments without additional security hardening.

- Runs with root privileges
- Minimal security isolation
- No user namespace support
- Basic network security
- No seccomp or AppArmor integration

## Acknowledgments

- **Vishvananda/netlink**: For Go network interface management

## License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.

---

<p align="center">
    Made with ❤️ by <a href="https://github.com/yashraj-n">Yashraj Narke</a>
</p>
