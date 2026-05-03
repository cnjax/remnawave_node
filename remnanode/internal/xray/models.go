package xray

// SystemInfo represents system information
type SystemInfo struct {
	CPUCores    int    `json:"cpuCores"`
	CPUModel    string `json:"cpuModel"`
	MemoryTotal string `json:"memoryTotal"`
}

// NodeInformation represents node information
type NodeInformation struct {
	Version string `json:"version"`
}

// StartXrayResponse represents the response from starting Xray
type StartXrayResponse struct {
	IsStarted         bool             `json:"isStarted"`
	Version           *string          `json:"version"`
	Error             *string          `json:"error"`
	SystemInformation *SystemInfo      `json:"systemInformation"`
	NodeInformation   *NodeInformation `json:"nodeInformation"`
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
