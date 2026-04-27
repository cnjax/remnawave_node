package sysinfo

import (
	"fmt"
	"runtime"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// SystemInfo holds static system information
type SystemInfo struct {
	Arch             string   `json:"arch"`
	CPUs             int      `json:"cpus"`
	CPUModel         string   `json:"cpuModel"`
	MemoryTotal      uint64   `json:"memoryTotal"`
	Hostname         string   `json:"hostname"`
	Platform         string   `json:"platform"`
	Release          string   `json:"release"`
	Type             string   `json:"type"`
	Version          string   `json:"version"`
	NetworkInterfaces []string `json:"networkInterfaces"`
}

// NetworkInterfaceStats holds per-interface traffic rates
type NetworkInterfaceStats struct {
	Interface    string  `json:"interface"`
	RxBytesPerSec float64 `json:"rxBytesPerSec"`
	TxBytesPerSec float64 `json:"txBytesPerSec"`
	RxTotal      uint64  `json:"rxTotal"`
	TxTotal      uint64  `json:"txTotal"`
}

// SystemStats holds dynamic system statistics
type SystemStats struct {
	MemoryFree uint64                 `json:"memoryFree"`
	MemoryUsed uint64                 `json:"memoryUsed"`
	Uptime     uint64                 `json:"uptime"`
	LoadAvg    []float64              `json:"loadAvg"`
	Interface  *NetworkInterfaceStats `json:"interface"`
}

// GetSystemInfo retrieves static system information
func GetSystemInfo() (*SystemInfo, error) {
	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU info: %w", err)
	}

	cpuCount, err := cpu.Counts(true)
	if err != nil {
		cpuCount = len(cpuInfo)
	}

	cpuModel := "Unknown"
	if len(cpuInfo) > 0 {
		cpuModel = cpuInfo[0].ModelName
	}

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}

	hostInfo, _ := host.Info()
	hostname := ""
	platform := runtime.GOOS
	release := ""
	osType := runtime.GOOS
	version := ""

	if hostInfo != nil {
		hostname = hostInfo.Hostname
		platform = hostInfo.Platform
		release = hostInfo.KernelVersion
		osType = hostInfo.OS
		version = hostInfo.PlatformVersion
	}

	netIfaces, _ := net.Interfaces()
	ifaceNames := make([]string, 0, len(netIfaces))
	for _, iface := range netIfaces {
		ifaceNames = append(ifaceNames, iface.Name)
	}

	return &SystemInfo{
		Arch:              runtime.GOARCH,
		CPUs:              cpuCount,
		CPUModel:          cpuModel,
		MemoryTotal:       memInfo.Total,
		Hostname:          hostname,
		Platform:          platform,
		Release:           release,
		Type:              osType,
		Version:           version,
		NetworkInterfaces: ifaceNames,
	}, nil
}

// GetSystemStats retrieves dynamic system statistics
func GetSystemStats() (*SystemStats, error) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}

	uptimeInfo, _ := host.Uptime()

	loadAvg := []float64{0, 0, 0}
	if avgStat, err := load.Avg(); err == nil {
		loadAvg = []float64{avgStat.Load1, avgStat.Load5, avgStat.Load15}
	}

	return &SystemStats{
		MemoryFree: memInfo.Free,
		MemoryUsed: memInfo.Used,
		Uptime:     uptimeInfo,
		LoadAvg:    loadAvg,
		Interface:  nil,
	}, nil
}
