package httpcheck

type Timings struct {
	DNSMS     int64 `json:"dnsMs"`
	ConnectMS int64 `json:"connectMs"`
	TLSMS     int64 `json:"tlsMs"`
	TTFBMS    int64 `json:"ttfbMs"`
	TotalMS   int64 `json:"totalMs"`
}

type Redirect struct {
	URL        string `json:"url"`
	StatusCode int    `json:"statusCode"`
}

type Result struct {
	RequestedURL    string              `json:"requestedUrl"`
	FinalURL        string              `json:"finalUrl,omitempty"`
	Method          string              `json:"method"`
	StatusCode      int                 `json:"statusCode,omitempty"`
	Protocol        string              `json:"protocol,omitempty"`
	ResponseHeaders map[string][]string `json:"responseHeaders,omitempty"`
	ContentType     string              `json:"contentType,omitempty"`
	ContentLength   int64               `json:"contentLength,omitempty"`
	Redirects       []Redirect          `json:"redirectChain"`
	ResolvedIP      string              `json:"resolvedIp,omitempty"`
	RemoteAddress   string              `json:"remoteAddress,omitempty"`
	Timings         Timings             `json:"timings"`
	BodySHA256      string              `json:"bodySha256,omitempty"`
	BodyPreview     string              `json:"bodyPreview,omitempty"`
	BodyTruncated   bool                `json:"bodyTruncated"`
}
