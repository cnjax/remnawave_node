package plugin

// SyncRequest matches the panel contract for POST /node/plugin/sync.
type SyncRequest struct {
	Plugin *PluginPayload `json:"plugin"`
}

type PluginPayload struct {
	Config map[string]interface{} `json:"config"`
	UUID   string                 `json:"uuid"`
	Name   string                 `json:"name"`
}

type SyncResponse struct {
	Accepted bool `json:"accepted"`
}
