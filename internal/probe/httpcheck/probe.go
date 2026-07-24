package httpcheck

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/XxKotfeJxX/netscope/internal/network"
	"github.com/XxKotfeJxX/netscope/internal/target"
	"github.com/google/uuid"
)

const maxBodyBytes = 64 * 1024

type Probe struct {
	dialer *network.SecureDialer
}

func New(dialer *network.SecureDialer) *Probe {
	return &Probe{dialer: dialer}
}

func (p *Probe) Type() diagnostics.CheckType {
	return diagnostics.CheckHTTP
}

func (p *Probe) Run(
	ctx context.Context,
	parsedTarget target.Target,
	options diagnostics.RunOptions,
) diagnostics.CheckResult {
	started := time.Now().UTC()
	requestedURL := targetURL(parsedTarget)
	data := Result{
		RequestedURL: requestedURL,
		Method:       options.HTTPMethod,
		Redirects:    []Redirect{},
	}
	if data.Method == "" {
		data.Method = http.MethodGet
	}

	dnsStarted := time.Now()
	if addresses, err := p.dialer.Resolve(ctx, parsedTarget.Host); err == nil {
		data.Timings.DNSMS = time.Since(dnsStarted).Milliseconds()
		if len(addresses) > 0 {
			data.ResolvedIP = addresses[0].String()
		}
	} else {
		return failedResult(started, data, classifyRequestError(ctx, err))
	}

	var timingMutex sync.Mutex
	var connectStarted time.Time
	var tlsStarted time.Time
	trace := &httptrace.ClientTrace{
		ConnectStart: func(_, _ string) {
			timingMutex.Lock()
			connectStarted = time.Now()
			timingMutex.Unlock()
		},
		ConnectDone: func(_, _ string, _ error) {
			timingMutex.Lock()
			if !connectStarted.IsZero() {
				data.Timings.ConnectMS += time.Since(connectStarted).Milliseconds()
			}
			timingMutex.Unlock()
		},
		TLSHandshakeStart: func() {
			timingMutex.Lock()
			tlsStarted = time.Now()
			timingMutex.Unlock()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			timingMutex.Lock()
			if !tlsStarted.IsZero() {
				data.Timings.TLSMS += time.Since(tlsStarted).Milliseconds()
			}
			timingMutex.Unlock()
		},
		GotFirstResponseByte: func() {
			timingMutex.Lock()
			data.Timings.TTFBMS = time.Since(started).Milliseconds()
			timingMutex.Unlock()
		},
		GotConn: func(info httptrace.GotConnInfo) {
			timingMutex.Lock()
			data.RemoteAddress = info.Conn.RemoteAddr().String()
			timingMutex.Unlock()
		},
	}

	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           p.dialer.DialContext,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   time.Duration(options.TimeoutMS) * time.Millisecond,
		ResponseHeaderTimeout: time.Duration(options.TimeoutMS) * time.Millisecond,
		IdleConnTimeout:       30 * time.Second,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) > 0 && via[len(via)-1].Response != nil {
				data.Redirects = append(data.Redirects, Redirect{
					URL: via[len(via)-1].URL.String(), StatusCode: via[len(via)-1].Response.StatusCode,
				})
			}
			if !options.FollowRedirects {
				return http.ErrUseLastResponse
			}
			if len(via) > options.MaxRedirects {
				return errors.New("redirect limit exceeded")
			}
			if request.URL.Scheme != "http" && request.URL.Scheme != "https" {
				return errors.New("redirect scheme is not allowed")
			}
			if request.URL.User != nil {
				return target.ErrCredentials
			}
			return nil
		},
	}

	request, err := http.NewRequestWithContext(
		httptrace.WithClientTrace(ctx, trace),
		data.Method,
		requestedURL,
		nil,
	)
	if err != nil {
		return failedResult(started, data, requestFailure{
			code: "invalid_target", message: "The HTTP target URL is invalid.",
		})
	}
	request.Header.Set("User-Agent", "NetScope/0.1")
	response, err := client.Do(request)
	if err != nil {
		return failedResult(started, data, classifyRequestError(ctx, err))
	}
	defer func() { _ = response.Body.Close() }()

	data.FinalURL = response.Request.URL.String()
	data.StatusCode = response.StatusCode
	data.Protocol = response.Proto
	data.ResponseHeaders = safeHeaders(response.Header)
	data.ContentType = response.Header.Get("Content-Type")
	data.ContentLength = response.ContentLength
	body, readErr := io.ReadAll(io.LimitReader(response.Body, maxBodyBytes+1))
	if readErr != nil && ctx.Err() != nil {
		return failedResult(started, data, classifyRequestError(ctx, readErr))
	}
	if len(body) > maxBodyBytes {
		body = body[:maxBodyBytes]
		data.BodyTruncated = true
	}
	hash := sha256.Sum256(body)
	data.BodySHA256 = hex.EncodeToString(hash[:])
	if isTextContent(data.ContentType) {
		preview := body
		if len(preview) > 512 {
			preview = preview[:512]
		}
		data.BodyPreview = strings.ToValidUTF8(string(preview), "�")
	}
	data.Timings.TotalMS = time.Since(started).Milliseconds()

	status := diagnostics.CheckPassed
	summary := "HTTP request succeeded"
	switch {
	case response.StatusCode >= 500:
		status = diagnostics.CheckFailed
		summary = "Server returned an error response"
	case response.StatusCode >= 400:
		status = diagnostics.CheckWarning
		summary = "Target returned a client error response"
	}
	encoded, _ := json.Marshal(data)
	completed := time.Now().UTC()
	return diagnostics.CheckResult{
		ID: uuid.New(), Type: diagnostics.CheckHTTP, Status: status,
		DurationMS: completed.Sub(started).Milliseconds(), Summary: summary,
		Data: encoded, StartedAt: started, CompletedAt: completed,
	}
}

type requestFailure struct {
	code    string
	message string
}

func failedResult(
	started time.Time,
	data Result,
	failure requestFailure,
) diagnostics.CheckResult {
	completed := time.Now().UTC()
	data.Timings.TotalMS = completed.Sub(started).Milliseconds()
	encoded, _ := json.Marshal(data)
	status := diagnostics.CheckFailed
	if failure.code == "probe_cancelled" {
		status = diagnostics.CheckCancelled
	}
	return diagnostics.CheckResult{
		ID: uuid.New(), Type: diagnostics.CheckHTTP, Status: status,
		DurationMS: data.Timings.TotalMS, Summary: "HTTP request failed", Data: encoded,
		ErrorCode: failure.code, ErrorMessage: failure.message,
		StartedAt: started, CompletedAt: completed,
	}
}

func classifyRequestError(ctx context.Context, err error) requestFailure {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded), errors.Is(err, context.DeadlineExceeded):
		return requestFailure{code: "probe_timeout", message: "The HTTP request timed out."}
	case errors.Is(ctx.Err(), context.Canceled), errors.Is(err, context.Canceled):
		return requestFailure{code: "probe_cancelled", message: "The HTTP request was cancelled."}
	case errors.Is(err, target.ErrAddressBlocked):
		return requestFailure{code: "target_blocked", message: "The redirect target is blocked by network policy."}
	}
	var urlError *url.Error
	if errors.As(err, &urlError) {
		var networkError net.Error
		if errors.As(urlError.Err, &networkError) && networkError.Timeout() {
			return requestFailure{code: "probe_timeout", message: "The HTTP request timed out."}
		}
	}
	return requestFailure{code: "http_error", message: "The HTTP request could not be completed."}
}

func targetURL(parsed target.Target) string {
	if parsed.NormalizedURL != "" {
		return parsed.NormalizedURL
	}
	host := parsed.Host
	if parsed.Address.Is6() {
		host = "[" + host + "]"
	}
	if parsed.Port > 0 {
		host = network.PortAddress(parsed.Host, parsed.Port)
	}
	return (&url.URL{Scheme: "https", Host: host, Path: parsed.Path}).String()
}

func safeHeaders(headers http.Header) map[string][]string {
	result := make(map[string][]string, len(headers))
	for name, values := range headers {
		if strings.EqualFold(name, "Set-Cookie") || strings.EqualFold(name, "Proxy-Authenticate") {
			continue
		}
		result[name] = append([]string(nil), values...)
	}
	return result
}

func isTextContent(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = strings.Split(contentType, ";")[0]
	}
	return strings.HasPrefix(mediaType, "text/") ||
		strings.Contains(mediaType, "json") ||
		strings.Contains(mediaType, "xml")
}
