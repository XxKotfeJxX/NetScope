package tcp

type PortResult struct {
	Port          int    `json:"port"`
	Status        string `json:"status"`
	ResolvedIP    string `json:"resolvedIp,omitempty"`
	ConnectTimeMS int64  `json:"connectTimeMs"`
	ErrorCode     string `json:"errorCode,omitempty"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
}

type Result struct {
	Ports []PortResult `json:"ports"`
}
