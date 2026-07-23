package tlscheck

import "time"

type Certificate struct {
	Subject      string    `json:"subject"`
	Issuer       string    `json:"issuer"`
	SerialNumber string    `json:"serialNumber"`
	DNSNames     []string  `json:"dnsNames"`
	IPAddresses  []string  `json:"ipAddresses"`
	ValidFrom    time.Time `json:"validFrom"`
	ValidUntil   time.Time `json:"validUntil"`
}

type Result struct {
	TLSVersion    string        `json:"tlsVersion,omitempty"`
	CipherSuite   string        `json:"cipherSuite,omitempty"`
	ServerName    string        `json:"serverName"`
	ResolvedIP    string        `json:"resolvedIp,omitempty"`
	Subject       string        `json:"subject,omitempty"`
	Issuer        string        `json:"issuer,omitempty"`
	SerialNumber  string        `json:"serialNumber,omitempty"`
	SANs          []string      `json:"sans"`
	ValidFrom     time.Time     `json:"validFrom,omitempty"`
	ValidUntil    time.Time     `json:"validUntil,omitempty"`
	DaysRemaining int           `json:"daysRemaining"`
	HostnameValid bool          `json:"hostnameValid"`
	ChainValid    bool          `json:"chainValid"`
	SelfSigned    bool          `json:"selfSigned"`
	Expired       bool          `json:"expired"`
	Chain         []Certificate `json:"certificateChain"`
	Warnings      []string      `json:"warnings"`
}
