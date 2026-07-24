package monitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/network"
	"github.com/google/uuid"
)

type NotificationRepository interface {
	ListNotificationChannels(context.Context, uuid.UUID) ([]NotificationChannel, error)
}

type WebhookDelivery interface {
	Deliver(context.Context, string, Alert) error
}

type EmailDelivery interface {
	Deliver(context.Context, string, Alert) error
}

type ChannelNotifier struct {
	repository NotificationRepository
	webhooks   WebhookDelivery
	email      EmailDelivery
}

func NewChannelNotifier(
	repository NotificationRepository,
	webhooks WebhookDelivery,
	email EmailDelivery,
) *ChannelNotifier {
	return &ChannelNotifier{
		repository: repository,
		webhooks:   webhooks,
		email:      email,
	}
}

func (n *ChannelNotifier) Notify(ctx context.Context, alert Alert) error {
	channels, err := n.repository.ListNotificationChannels(ctx, alert.Target.ID)
	if err != nil {
		return fmt.Errorf("list notification channels: %w", err)
	}

	var deliveryErrors []error
	for _, channel := range channels {
		if !channel.Enabled {
			continue
		}

		switch channel.Kind {
		case NotificationWebhook:
			if n.webhooks == nil {
				err = errors.New("webhook delivery is not configured")
			} else {
				err = n.webhooks.Deliver(ctx, channel.Destination, alert)
			}
		case NotificationEmail:
			if n.email == nil {
				err = errors.New("email delivery is not configured")
			} else {
				err = n.email.Deliver(ctx, channel.Destination, alert)
			}
		default:
			err = fmt.Errorf("unsupported notification kind %q", channel.Kind)
		}
		if err != nil {
			deliveryErrors = append(
				deliveryErrors,
				fmt.Errorf("%s channel %s: %w", channel.Kind, channel.ID, err),
			)
		}
	}
	return errors.Join(deliveryErrors...)
}

type WebhookSender struct {
	client    *http.Client
	userAgent string
}

func NewWebhookSender(
	dialer *network.SecureDialer,
	timeout time.Duration,
	version string,
) *WebhookSender {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if strings.TrimSpace(version) == "" {
		version = "dev"
	}
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		IdleConnTimeout:       30 * time.Second,
	}
	return &WebhookSender{
		client: &http.Client{
			Transport: transport,
			Timeout:   timeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return errors.New("webhook redirects are not allowed")
			},
		},
		userAgent: "NetScope/" + version,
	}
}

func (s *WebhookSender) Deliver(
	ctx context.Context,
	destination string,
	alert Alert,
) error {
	parsed, err := url.Parse(destination)
	if err != nil || parsed.Host == "" ||
		(parsed.Scheme != "https" && parsed.Scheme != "http") {
		return errors.New("invalid webhook URL")
	}
	if parsed.User != nil {
		return errors.New("webhook URL credentials are not allowed")
	}

	payload, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("encode webhook payload: %w", err)
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		parsed.String(),
		bytes.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", s.userAgent)

	response, err := s.client.Do(request)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode < http.StatusOK ||
		response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("webhook returned HTTP %d", response.StatusCode)
	}
	return nil
}
