package dns

type MXRecord struct {
	Host       string `json:"host"`
	Preference uint16 `json:"preference"`
}

type RecordError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Result struct {
	A      []string               `json:"a"`
	AAAA   []string               `json:"aaaa"`
	CNAME  string                 `json:"cname,omitempty"`
	MX     []MXRecord             `json:"mx"`
	NS     []string               `json:"ns"`
	TXT    []string               `json:"txt"`
	PTR    []string               `json:"ptr"`
	Errors map[string]RecordError `json:"errors,omitempty"`
}
