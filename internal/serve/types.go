package serve

// ServerInfo is the JSON response for a server listing entry.
type ServerInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ToolInfo is the JSON response for a tool listing/detail entry.
type ToolInfo struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  []ParamInfo `json:"parameters"`
}

// ParamInfo describes a single tool parameter.
type ParamInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// ErrorResponse is the standard JSON error response.
type ErrorResponse struct {
	Error string `json:"error"`
}
