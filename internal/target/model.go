package target

import "net/netip"

type Kind string

const (
	KindURL      Kind = "url"
	KindHostname Kind = "hostname"
	KindIPv4     Kind = "ipv4"
	KindIPv6     Kind = "ipv6"
)

type Target struct {
	Input         string     `json:"input"`
	Kind          Kind       `json:"kind"`
	Host          string     `json:"host"`
	Scheme        string     `json:"scheme,omitempty"`
	Port          int        `json:"port,omitempty"`
	Path          string     `json:"path,omitempty"`
	NormalizedURL string     `json:"normalizedUrl,omitempty"`
	Address       netip.Addr `json:"-"`
}

func (t Target) IsIP() bool {
	return t.Address.IsValid()
}
