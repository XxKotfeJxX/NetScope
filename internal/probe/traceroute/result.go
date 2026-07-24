package traceroute

type Hop struct {
	Number      int     `json:"number"`
	Status      string  `json:"status"`
	Address     string  `json:"address,omitempty"`
	RTTMS       float64 `json:"rttMs,omitempty"`
	Destination bool    `json:"destination"`
}

type Result struct {
	Address string `json:"address"`
	Reached bool   `json:"reached"`
	MaxHops int    `json:"maxHops"`
	Hops    []Hop  `json:"hops"`
}
