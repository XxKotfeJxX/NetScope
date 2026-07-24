package tlscheck

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/XxKotfeJxX/netscope/internal/network"
	"github.com/XxKotfeJxX/netscope/internal/target"
	"github.com/google/uuid"
)

type Probe struct {
	dialer *network.SecureDialer
}

func New(dialer *network.SecureDialer) *Probe {
	return &Probe{dialer: dialer}
}

func (p *Probe) Type() diagnostics.CheckType {
	return diagnostics.CheckTLS
}

func (p *Probe) Run(
	ctx context.Context,
	parsedTarget target.Target,
	options diagnostics.RunOptions,
) diagnostics.CheckResult {
	started := time.Now().UTC()
	ctx = network.WithIPVersion(ctx, options.IPVersion)
	port := parsedTarget.Port
	if port == 0 {
		port = 443
	}
	data := Result{
		ServerName: parsedTarget.Host, SANs: []string{},
		Chain: []Certificate{}, Warnings: []string{},
	}

	state, resolvedIP, verifiedError := p.handshake(ctx, parsedTarget.Host, port, true)
	if verifiedError != nil && ctx.Err() == nil {
		// A second diagnostic connection collects the peer certificate. This
		// connection is never treated as trusted; hostname, dates, and chain are
		// evaluated explicitly below.
		state, resolvedIP, _ = p.handshake(ctx, parsedTarget.Host, port, false)
	}
	data.ResolvedIP = resolvedIP
	if len(state.PeerCertificates) == 0 {
		return failedTLSResult(started, data, classifyTLSError(ctx, verifiedError))
	}

	leaf := state.PeerCertificates[0]
	data.TLSVersion = tlsVersionName(state.Version)
	data.CipherSuite = tls.CipherSuiteName(state.CipherSuite)
	data.Subject = leaf.Subject.String()
	data.Issuer = leaf.Issuer.String()
	data.SerialNumber = leaf.SerialNumber.String()
	data.ValidFrom = leaf.NotBefore
	data.ValidUntil = leaf.NotAfter
	data.DaysRemaining = int(time.Until(leaf.NotAfter).Hours() / 24)
	data.Expired = time.Now().Before(leaf.NotBefore) || time.Now().After(leaf.NotAfter)
	data.SelfSigned = bytes.Equal(leaf.RawIssuer, leaf.RawSubject) && leaf.CheckSignatureFrom(leaf) == nil
	data.SANs = append(data.SANs, leaf.DNSNames...)
	for _, address := range leaf.IPAddresses {
		data.SANs = append(data.SANs, address.String())
	}
	data.HostnameValid = leaf.VerifyHostname(parsedTarget.Host) == nil
	data.ChainValid = verifyChain(leaf, state.PeerCertificates[1:], parsedTarget.Host) == nil
	for _, certificate := range state.PeerCertificates {
		item := Certificate{
			Subject: certificate.Subject.String(), Issuer: certificate.Issuer.String(),
			SerialNumber: certificate.SerialNumber.String(),
			DNSNames:     append([]string(nil), certificate.DNSNames...),
			ValidFrom:    certificate.NotBefore, ValidUntil: certificate.NotAfter,
		}
		for _, address := range certificate.IPAddresses {
			item.IPAddresses = append(item.IPAddresses, address.String())
		}
		data.Chain = append(data.Chain, item)
	}

	status := diagnostics.CheckPassed
	errorCode := ""
	errorMessage := ""
	summary := "TLS certificate is valid"
	switch {
	case data.Expired:
		status = diagnostics.CheckFailed
		errorCode = "tls_expired"
		errorMessage = "The TLS certificate is expired or not yet valid."
		summary = "TLS certificate date is invalid"
	case !data.HostnameValid:
		status = diagnostics.CheckFailed
		errorCode = "tls_hostname_mismatch"
		errorMessage = "The TLS certificate does not match the target hostname."
		summary = "TLS hostname validation failed"
	case !data.ChainValid:
		status = diagnostics.CheckFailed
		errorCode = "tls_untrusted"
		errorMessage = "The TLS certificate chain is not trusted."
		summary = "TLS certificate chain is not trusted"
	case data.DaysRemaining < 7:
		status = diagnostics.CheckWarning
		data.Warnings = append(data.Warnings, "certificate_expires_within_7_days")
		summary = "TLS certificate expires soon"
	case data.DaysRemaining < 30:
		status = diagnostics.CheckWarning
		data.Warnings = append(data.Warnings, "certificate_expires_within_30_days")
		summary = "TLS certificate expires within 30 days"
	}
	if data.SelfSigned {
		data.Warnings = append(data.Warnings, "self_signed_certificate")
	}
	if len(data.Chain) < 2 {
		data.Warnings = append(data.Warnings, "incomplete_chain")
	}

	completed := time.Now().UTC()
	encoded, _ := json.Marshal(data)
	return diagnostics.CheckResult{
		ID: uuid.New(), Type: diagnostics.CheckTLS, Status: status,
		DurationMS: completed.Sub(started).Milliseconds(), Summary: summary, Data: encoded,
		ErrorCode: errorCode, ErrorMessage: errorMessage,
		StartedAt: started, CompletedAt: completed,
	}
}

func (p *Probe) handshake(
	ctx context.Context,
	host string,
	port int,
	verify bool,
) (tls.ConnectionState, string, error) {
	rawConnection, err := p.dialer.DialContext(ctx, "tcp", network.PortAddress(host, port))
	if err != nil {
		return tls.ConnectionState{}, "", err
	}
	resolvedIP := network.AddressFromConnection(rawConnection)
	config := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: host,
	}
	if !verify {
		config.InsecureSkipVerify = true // Diagnostic inspection only; manually verified by Run.
	}
	connection := tls.Client(rawConnection, config)
	defer func() { _ = connection.Close() }()
	if err := connection.HandshakeContext(ctx); err != nil {
		return connection.ConnectionState(), resolvedIP, err
	}
	return connection.ConnectionState(), resolvedIP, nil
}

func verifyChain(leaf *x509.Certificate, chain []*x509.Certificate, host string) error {
	intermediates := x509.NewCertPool()
	for _, certificate := range chain {
		intermediates.AddCert(certificate)
	}
	_, err := leaf.Verify(x509.VerifyOptions{
		DNSName: host, Intermediates: intermediates,
	})
	return err
}

type tlsFailure struct {
	code    string
	message string
}

func failedTLSResult(
	started time.Time,
	data Result,
	failure tlsFailure,
) diagnostics.CheckResult {
	completed := time.Now().UTC()
	encoded, _ := json.Marshal(data)
	status := diagnostics.CheckFailed
	if failure.code == "probe_cancelled" {
		status = diagnostics.CheckCancelled
	}
	return diagnostics.CheckResult{
		ID: uuid.New(), Type: diagnostics.CheckTLS, Status: status,
		DurationMS: completed.Sub(started).Milliseconds(), Summary: "TLS handshake failed",
		Data: encoded, ErrorCode: failure.code, ErrorMessage: failure.message,
		StartedAt: started, CompletedAt: completed,
	}
}

func classifyTLSError(ctx context.Context, err error) tlsFailure {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded), errors.Is(err, context.DeadlineExceeded):
		return tlsFailure{code: "probe_timeout", message: "The TLS handshake timed out."}
	case errors.Is(ctx.Err(), context.Canceled), errors.Is(err, context.Canceled):
		return tlsFailure{code: "probe_cancelled", message: "The TLS check was cancelled."}
	}
	var hostnameError x509.HostnameError
	if errors.As(err, &hostnameError) {
		return tlsFailure{
			code:    "tls_hostname_mismatch",
			message: "The TLS certificate does not match the target hostname.",
		}
	}
	var certificateError x509.CertificateInvalidError
	if errors.As(err, &certificateError) {
		return tlsFailure{code: "tls_invalid", message: "The TLS certificate is invalid."}
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return tlsFailure{code: "probe_timeout", message: "The TLS handshake timed out."}
	}
	return tlsFailure{code: "tls_handshake_failed", message: "The TLS handshake could not be completed."}
}

func tlsVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS10:
		return "TLS 1.0"
	default:
		return fmt.Sprintf("0x%04x", version)
	}
}
