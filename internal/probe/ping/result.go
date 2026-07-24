package ping

type Sample struct {
	Sequence int     `json:"sequence"`
	Status   string  `json:"status"`
	Address  string  `json:"address,omitempty"`
	RTTMS    float64 `json:"rttMs,omitempty"`
}

type Result struct {
	Address           string   `json:"address"`
	PacketsSent       int      `json:"packetsSent"`
	PacketsReceived   int      `json:"packetsReceived"`
	PacketLossPercent float64  `json:"packetLossPercent"`
	MinRTTMS          float64  `json:"minRttMs"`
	AverageRTTMS      float64  `json:"averageRttMs"`
	MaxRTTMS          float64  `json:"maxRttMs"`
	Samples           []Sample `json:"samples"`
}
