package sysinfo

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

// SystemInfo holds static system information (read once at startup).
type SystemInfo struct {
	Arch              string   `json:"arch"`
	CPUs              int      `json:"cpus"`
	CPUModel          string   `json:"cpuModel"`
	MemoryTotal       uint64   `json:"memoryTotal"`
	Hostname          string   `json:"hostname"`
	Platform          string   `json:"platform"`
	Release           string   `json:"release"`
	Type              string   `json:"type"`
	Version           string   `json:"version"`
	NetworkInterfaces []string `json:"networkInterfaces"`
}

// NetworkInterfaceStats holds per-second traffic rates for one interface.
type NetworkInterfaceStats struct {
	Interface     string  `json:"interface"`
	RxBytesPerSec float64 `json:"rxBytesPerSec"`
	TxBytesPerSec float64 `json:"txBytesPerSec"`
	RxTotal       uint64  `json:"rxTotal"`
	TxTotal       uint64  `json:"txTotal"`
}

// SystemStats holds dynamic system statistics (polled on demand).
type SystemStats struct {
	MemoryFree uint64                 `json:"memoryFree"`
	MemoryUsed uint64                 `json:"memoryUsed"`
	Uptime     uint64                 `json:"uptime"`
	LoadAvg    []float64              `json:"loadAvg"`
	Interface  *NetworkInterfaceStats `json:"interface"`
}

// GetSystemInfo reads static system information.
func GetSystemInfo() (*SystemInfo, error) {
	cpuInfo, _ := cpu.Info()
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
	hostname, platform, release, osType, version := "", runtime.GOOS, "", runtime.GOOS, ""
	if hostInfo != nil {
		hostname = hostInfo.Hostname
		platform = hostInfo.Platform
		release = hostInfo.KernelVersion
		osType = hostInfo.OS
		version = hostInfo.PlatformVersion
	}

	ifaceNames := readNetDevNames()

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

// GetSystemStats reads dynamic system statistics including network rates.
func GetSystemStats() (*SystemStats, error) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}

	uptime, _ := host.Uptime()

	loadAvg := []float64{0, 0, 0}
	if avg, err := load.Avg(); err == nil {
		loadAvg = []float64{avg.Load1, avg.Load5, avg.Load15}
	}

	return &SystemStats{
		MemoryFree: memInfo.Free,
		MemoryUsed: memInfo.Used,
		Uptime:     uptime,
		LoadAvg:    loadAvg,
		Interface:  netSampler.get(),
	}, nil
}

// ---- Network sampler --------------------------------------------------------

type netSnapshot struct {
	rxBytes uint64
	txBytes uint64
	ts      time.Time
}

type networkSampler struct {
	mu           sync.RWMutex
	prev         map[string]netSnapshot
	current      *NetworkInterfaceStats
	defaultIface string
}

var netSampler = &networkSampler{}

func init() {
	netSampler.defaultIface = resolveDefaultInterface()
	netSampler.prev = readNetDev()
	go netSampler.run()
}

func (ns *networkSampler) run() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		ns.tick()
	}
}

func (ns *networkSampler) tick() {
	now := readNetDev()
	iface := ns.defaultIface

	ns.mu.Lock()
	defer ns.mu.Unlock()

	prev := ns.prev
	ns.prev = now

	if iface == "" || prev == nil {
		return
	}

	p, okP := prev[iface]
	c, okC := now[iface]
	if !okP || !okC {
		return
	}

	elapsed := c.ts.Sub(p.ts).Seconds()
	if elapsed <= 0 {
		return
	}

	ns.current = &NetworkInterfaceStats{
		Interface:     iface,
		RxBytesPerSec: max0(float64(c.rxBytes-p.rxBytes) / elapsed),
		TxBytesPerSec: max0(float64(c.txBytes-p.txBytes) / elapsed),
		RxTotal:       c.rxBytes,
		TxTotal:       c.txBytes,
	}
}

func (ns *networkSampler) get() *NetworkInterfaceStats {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	return ns.current
}

func max0(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}

// ---- /proc helpers ----------------------------------------------------------

// readNetDev parses /proc/net/dev and returns rx/tx byte totals per interface.
func readNetDev() map[string]netSnapshot {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil
	}
	defer f.Close()

	result := make(map[string]netSnapshot)
	now := time.Now()
	scanner := bufio.NewScanner(f)

	// Skip 2 header lines
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) < 10 {
			continue
		}
		iface := strings.TrimSuffix(parts[0], ":")
		rxBytes, _ := strconv.ParseUint(parts[1], 10, 64)
		txBytes, _ := strconv.ParseUint(parts[9], 10, 64)
		result[iface] = netSnapshot{rxBytes: rxBytes, txBytes: txBytes, ts: now}
	}
	return result
}

// readNetDevNames returns interface names listed in /proc/net/dev.
func readNetDevNames() []string {
	m := readNetDev()
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	return names
}

// resolveDefaultInterface reads /proc/net/route and returns the interface for the default route.
func resolveDefaultInterface() string {
	f, err := os.Open("/proc/net/route")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Scan() // skip header
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[1] == "00000000" {
			return fields[0]
		}
	}
	return ""
}
