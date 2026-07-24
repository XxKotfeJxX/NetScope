package target

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/net/idna"
)

const MaxLength = 2048

var (
	ErrEmpty       = errors.New("target is required")
	ErrInvalid     = errors.New("target must be a valid hostname, IP address, or URL")
	ErrCredentials = errors.New("URL credentials are not allowed")
	ErrCIDR        = errors.New("CIDR targets are not allowed")
	ErrWildcard    = errors.New("wildcard hostnames are not allowed")
	ErrPort        = errors.New("port must be between 1 and 65535")
)

func Parse(input string) (Target, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return Target{}, ErrEmpty
	}
	if len(raw) > MaxLength || containsControl(raw) {
		return Target{}, ErrInvalid
	}
	if strings.Contains(raw, "*") {
		return Target{}, ErrWildcard
	}
	if _, _, err := net.ParseCIDR(raw); err == nil {
		return Target{}, ErrCIDR
	}

	if strings.Contains(raw, "://") {
		return parseURL(raw)
	}
	return parseHost(raw)
}

func parseURL(raw string) (Target, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return Target{}, ErrInvalid
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return Target{}, ErrInvalid
	}
	if parsed.User != nil {
		return Target{}, ErrCredentials
	}

	host, address, kind, err := normalizeHost(parsed.Hostname())
	if err != nil {
		return Target{}, err
	}
	port, err := parsePort(parsed.Port())
	if err != nil {
		return Target{}, err
	}
	if port == 0 {
		if parsed.Scheme == "https" {
			port = 443
		} else {
			port = 80
		}
	}

	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = net.JoinHostPort(host, strconv.Itoa(port))
	if (parsed.Scheme == "https" && port == 443) || (parsed.Scheme == "http" && port == 80) {
		parsed.Host = hostForURL(host, address)
	}
	parsed.Fragment = ""

	return Target{
		Input:         raw,
		Kind:          kindForURL(kind),
		Host:          host,
		Scheme:        parsed.Scheme,
		Port:          port,
		Path:          parsed.EscapedPath(),
		NormalizedURL: parsed.String(),
		Address:       address,
	}, nil
}

func parseHost(raw string) (Target, error) {
	hostPart := raw
	portPart := ""

	if strings.HasPrefix(raw, "[") {
		if strings.HasSuffix(raw, "]") {
			hostPart = strings.TrimSuffix(strings.TrimPrefix(raw, "["), "]")
		} else {
			var err error
			hostPart, portPart, err = net.SplitHostPort(raw)
			if err != nil {
				return Target{}, ErrInvalid
			}
		}
	} else if strings.Count(raw, ":") == 1 {
		var err error
		hostPart, portPart, err = net.SplitHostPort(raw)
		if err != nil {
			return Target{}, ErrInvalid
		}
	}

	host, address, kind, err := normalizeHost(hostPart)
	if err != nil {
		return Target{}, err
	}
	port, err := parsePort(portPart)
	if err != nil {
		return Target{}, err
	}

	return Target{
		Input:   raw,
		Kind:    kind,
		Host:    host,
		Port:    port,
		Address: address,
	}, nil
}

func normalizeHost(value string) (string, netip.Addr, Kind, error) {
	value = strings.TrimSuffix(strings.TrimSpace(value), ".")
	if value == "" {
		return "", netip.Addr{}, "", ErrInvalid
	}
	if address, err := netip.ParseAddr(value); err == nil {
		address = address.Unmap()
		kind := KindIPv6
		if address.Is4() {
			kind = KindIPv4
		}
		return address.String(), address, kind, nil
	}

	ascii, err := idna.Lookup.ToASCII(strings.ToLower(value))
	if err != nil || !validHostname(ascii) {
		return "", netip.Addr{}, "", ErrInvalid
	}
	return ascii, netip.Addr{}, KindHostname, nil
}

func validHostname(host string) bool {
	if len(host) == 0 || len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
				return false
			}
		}
	}
	return true
}

func parsePort(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("%w", ErrPort)
	}
	return port, nil
}

func hostForURL(host string, address netip.Addr) string {
	if address.Is6() {
		return "[" + host + "]"
	}
	return host
}

func kindForURL(_ Kind) Kind {
	return KindURL
}

func containsControl(value string) bool {
	for _, char := range value {
		if unicode.IsControl(char) {
			return true
		}
	}
	return false
}
