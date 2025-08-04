package main

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// SetupNetworking configures the container's network
func SetupNetworking(pid int, config *ContainerConfig) error {
	runtime.LockOSThread() // Required for network namespace operations
	defer runtime.UnlockOSThread()

	// Get host and container namespaces
	hostNs, err := netns.Get()
	if err != nil {
		return fmt.Errorf("failed to get host namespace: %v", err)
	}
	defer hostNs.Close()

	containerNs, err := netns.GetFromPid(pid)
	if err != nil {
		return fmt.Errorf("failed to get container namespace: %v", err)
	}
	defer containerNs.Close()

	// Create veth pair
	if err := createVethPair(); err != nil {
		return fmt.Errorf("failed to create veth pair: %v", err)
	}

	// Configure host side
	if err := configureHostNetwork(config); err != nil {
		return fmt.Errorf("failed to configure host network: %v", err)
	}

	// Move veth1 to container namespace
	veth1, err := netlink.LinkByName("veth1")
	if err != nil {
		return fmt.Errorf("failed to get veth1: %v", err)
	}

	if err := netlink.LinkSetNsFd(veth1, int(containerNs)); err != nil {
		return fmt.Errorf("failed to move veth1 to container: %v", err)
	}

	// Switch to container namespace and configure
	if err := netns.Set(containerNs); err != nil {
		return fmt.Errorf("failed to switch to container namespace: %v", err)
	}

	if err := configureContainerNetwork(config); err != nil {
		// Switch back to host namespace before returning error
		netns.Set(hostNs)
		return fmt.Errorf("failed to configure container network: %v", err)
	}

	// Switch back to host namespace
	if err := netns.Set(hostNs); err != nil {
		return fmt.Errorf("failed to switch back to host namespace: %v", err)
	}

	// Setup NAT and forwarding rules
	if err := setupNAT(config); err != nil {
		return fmt.Errorf("failed to setup NAT: %v", err)
	}

	return nil
}

// createVethPair creates a virtual ethernet pair
func createVethPair() error {
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{Name: "veth0"},
		PeerName:  "veth1",
	}
	
	if err := netlink.LinkAdd(veth); err != nil {
		return fmt.Errorf("failed to add veth pair: %v", err)
	}
	
	return nil
}

// configureHostNetwork configures the host side of the veth pair
func configureHostNetwork(config *ContainerConfig) error {
	veth0, err := netlink.LinkByName("veth0")
	if err != nil {
		return fmt.Errorf("failed to get veth0: %v", err)
	}

	// Bring up the interface
	if err := netlink.LinkSetUp(veth0); err != nil {
		return fmt.Errorf("failed to bring up veth0: %v", err)
	}

	// Parse and assign IP address
	hostIP := net.ParseIP(config.HostIP)
	if hostIP == nil {
		return fmt.Errorf("invalid host IP: %s", config.HostIP)
	}

	_, hostNet, err := net.ParseCIDR(config.NetworkCIDR)
	if err != nil {
		return fmt.Errorf("invalid network CIDR: %s", config.NetworkCIDR)
	}

	hostCIDR := &net.IPNet{IP: hostIP, Mask: hostNet.Mask}
	if err := netlink.AddrAdd(veth0, &netlink.Addr{IPNet: hostCIDR}); err != nil {
		return fmt.Errorf("failed to add address to veth0: %v", err)
	}

	return nil
}

// configureContainerNetwork configures the container side of the veth pair
func configureContainerNetwork(config *ContainerConfig) error {
	// Bring up loopback interface
	if err := bringUpLoopback(); err != nil {
		return fmt.Errorf("failed to bring up loopback: %v", err)
	}

	// Configure veth1
	veth1, err := netlink.LinkByName("veth1")
	if err != nil {
		return fmt.Errorf("failed to get veth1: %v", err)
	}

	// Bring up the interface
	if err := netlink.LinkSetUp(veth1); err != nil {
		return fmt.Errorf("failed to bring up veth1: %v", err)
	}

	// Parse and assign IP address
	containerIP := net.ParseIP(config.ContainerIP)
	if containerIP == nil {
		return fmt.Errorf("invalid container IP: %s", config.ContainerIP)
	}

	_, containerNet, err := net.ParseCIDR(config.NetworkCIDR)
	if err != nil {
		return fmt.Errorf("invalid network CIDR: %s", config.NetworkCIDR)
	}

	containerCIDR := &net.IPNet{IP: containerIP, Mask: containerNet.Mask}
	if err := netlink.AddrAdd(veth1, &netlink.Addr{IPNet: containerCIDR}); err != nil {
		return fmt.Errorf("failed to add address to veth1: %v", err)
	}

	// Add default route
	route := &netlink.Route{
		LinkIndex: veth1.Attrs().Index,
		Gw:        net.ParseIP(config.HostIP),
	}
	if err := netlink.RouteAdd(route); err != nil {
		return fmt.Errorf("failed to add default route: %v", err)
	}

	return nil
}

// bringUpLoopback brings up the loopback interface
func bringUpLoopback() error {
	lo, err := netlink.LinkByName("lo")
	if err != nil {
		return fmt.Errorf("failed to get loopback interface: %v", err)
	}
	
	if err := netlink.LinkSetUp(lo); err != nil {
		return fmt.Errorf("failed to bring up loopback: %v", err)
	}
	
	return nil
}

// setupNAT configures NAT and forwarding rules for container internet access
func setupNAT(config *ContainerConfig) error {
	_, network, err := net.ParseCIDR(config.NetworkCIDR)
	if err != nil {
		return fmt.Errorf("invalid network CIDR: %s", config.NetworkCIDR)
	}
	
	networkStr := network.String()
	
	// Enable masquerading for the container network
	cmd := exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", 
		"-s", networkStr, "-o", "eth0", "-j", "MASQUERADE")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to setup masquerading: %v", err)
	}

	// Allow forwarding for the container network
	cmd = exec.Command("iptables", "-A", "FORWARD", "-s", networkStr, "-j", "ACCEPT")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to setup forward rule (outbound): %v", err)
	}

	cmd = exec.Command("iptables", "-A", "FORWARD", "-d", networkStr, "-j", "ACCEPT")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to setup forward rule (inbound): %v", err)
	}

	return nil
}

// CleanupNetwork removes NAT and forwarding rules
func CleanupNetwork(config *ContainerConfig) {
	_, network, err := net.ParseCIDR(config.NetworkCIDR)
	if err != nil {
		return // Can't clean up if we can't parse the network
	}
	
	networkStr := network.String()
	
	// Remove NAT and forward rules (ignore errors as rules might not exist)
	exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING", 
		"-s", networkStr, "-o", "eth0", "-j", "MASQUERADE").Run()
	exec.Command("iptables", "-D", "FORWARD", "-s", networkStr, "-j", "ACCEPT").Run()
	exec.Command("iptables", "-D", "FORWARD", "-d", networkStr, "-j", "ACCEPT").Run()
}