package xray

import "github.com/remnawave/remnanode/pkg/sysinfo"

// NodeSystemInfo is the "info" sub-field of the system object.
type NodeSystemInfo struct {
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

// NodeSystem matches the TS TNodeSystem shape:
// { info: SystemInfo, stats: SystemStats, interface: NetworkInterfaceStats }
type NodeSystem struct {
	Info      *NodeSystemInfo                `json:"info"`
	Stats     *sysinfo.SystemStats           `json:"stats"`
	Interface *sysinfo.NetworkInterfaceStats `json:"interface"`
}

// NodeInformation represents node information
type NodeInformation struct {
	Version string `json:"version"`
}

// StartXrayResponse represents the response from starting Xray.
// Field name "system" matches TS contract (libs/contract/commands/xray/start.command.ts).
type StartXrayResponse struct {
	IsStarted       bool             `json:"isStarted"`
	Version         *string          `json:"version"`
	Error           *string          `json:"error"`
	System          *NodeSystem      `json:"system"`
	NodeInformation *NodeInformation `json:"nodeInformation"`
}

// StopXrayResponse represents the response from stopping Xray
type StopXrayResponse struct {
	IsStopped bool `json:"isStopped"`
}

// GetXrayStatusResponse represents the Xray status response
type GetXrayStatusResponse struct {
	IsRunning bool   `json:"isRunning"`
	Version   string `json:"version"`
}

// GetNodeHealthCheckResponse represents the node health check response.
// Field names match the TS contract (libs/contract/commands/xray/get-node-health-check.command.ts).
type GetNodeHealthCheckResponse struct {
	IsAlive                  bool   `json:"isAlive"`
	XrayInternalStatusCached bool   `json:"xrayInternalStatusCached"`
	XrayVersion              string `json:"xrayVersion"`
	NodeVersion              string `json:"nodeVersion"`
}
