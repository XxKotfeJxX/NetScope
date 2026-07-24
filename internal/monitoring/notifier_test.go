package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/network"
	"github.com/XxKotfeJxX/netscope/internal/target"
	"github.com/google/uuid"
)

type notificationRepositoryStub struct {
	channels []NotificationChannel
	err      error
}

func (r notificationRepositoryStub) ListNotificationChannels(
	context.Context,
	uuid.UUID,
) ([]NotificationChannel, error) {
	return r.channels, r.err
}

type deliveryRecorder struct {
	destinations []string
	err          error
}

func (d *deliveryRecorder) Deliver(
	_ context.Context,
	destination string,
	_ Alert,
) error {
	d.destinations = append(d.destinations, destination)
	return d.err
}

func TestChannelNotifierDeliversEnabledChannels(t *testing.T) {
	t.Parallel()

	targetID := uuid.New()
	webhooks := &deliveryRecorder{}
	email := &deliveryRecorder{}
	notifier := NewChannelNotifier(notificationRepositoryStub{channels: []NotificationChannel{
		{
			ID: uuid.New(), TargetID: targetID, Kind: NotificationWebhook,
			Destination: "https://hooks.example.test/netscope", Enabled: true,
		},
		{
			ID: uuid.New(), TargetID: targetID, Kind: NotificationEmail,
			Destination: "ops@example.test", Enabled: true,
		},
		{
			ID: uuid.New(), TargetID: targetID, Kind: NotificationWebhook,
			Destination: "https://disabled.example.test", Enabled: false,
		},
	}}, webhooks, email)

	if err := notifier.Notify(context.Background(), Alert{
		Target: Target{ID: targetID},
	}); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}
	if len(webhooks.destinations) != 1 ||
		webhooks.destinations[0] != "https://hooks.example.test/netscope" {
		t.Fatalf("webhook deliveries = %#v", webhooks.destinations)
	}
	if len(email.destinations) != 1 ||
		email.destinations[0] != "ops@example.test" {
		t.Fatalf("email deliveries = %#v", email.destinations)
	}
}

func TestChannelNotifierReturnsAllDeliveryFailures(t *testing.T) {
	t.Parallel()

	targetID := uuid.New()
	notifier := NewChannelNotifier(notificationRepositoryStub{channels: []NotificationChannel{
		{
			ID: uuid.New(), TargetID: targetID, Kind: NotificationWebhook,
			Destination: "https://hooks.example.test", Enabled: true,
		},
		{
			ID: uuid.New(), TargetID: targetID, Kind: NotificationEmail,
			Destination: "ops@example.test", Enabled: true,
		},
	}}, &deliveryRecorder{err: errors.New("webhook failed")}, nil)

	err := notifier.Notify(context.Background(), Alert{Target: Target{ID: targetID}})
	if err == nil ||
		!strings.Contains(err.Error(), "webhook failed") ||
		!strings.Contains(err.Error(), "email delivery is not configured") {
		t.Fatalf("Notify() error = %v", err)
	}
}

func TestWebhookSenderPostsAlert(t *testing.T) {
	t.Parallel()

	received := make(chan Alert, 1)
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", request.Method)
		}
		if request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content type = %q", request.Header.Get("Content-Type"))
		}
		if request.Header.Get("User-Agent") != "NetScope/0.3.0" {
			t.Errorf("user agent = %q", request.Header.Get("User-Agent"))
		}
		var alert Alert
		if err := json.NewDecoder(request.Body).Decode(&alert); err != nil {
			t.Errorf("decode webhook body: %v", err)
		}
		received <- alert
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	sender := NewWebhookSender(
		network.NewSecureDialer(
			nil,
			target.Policy{AllowLoopback: true},
			time.Second,
		),
		time.Second,
		"0.3.0",
	)
	alert := Alert{
		Target:  Target{ID: uuid.New(), Name: "Public API"},
		Status:  StatusUnavailable,
		Message: "Target failed 3 consecutive checks",
		SentAt:  time.Now().UTC(),
	}
	if err := sender.Deliver(context.Background(), server.URL, alert); err != nil {
		t.Fatalf("Deliver() error = %v", err)
	}
	if actual := <-received; actual.Target.ID != alert.Target.ID ||
		actual.Status != StatusUnavailable {
		t.Fatalf("delivered alert = %#v", actual)
	}
}

func TestWebhookSenderBlocksLoopbackUnderPublicPolicy(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(
		http.ResponseWriter,
		*http.Request,
	) {
		t.Error("blocked webhook reached server")
	}))
	defer server.Close()

	sender := NewWebhookSender(
		network.NewSecureDialer(nil, target.Policy{Public: true}, time.Second),
		time.Second,
		"test",
	)
	err := sender.Deliver(context.Background(), server.URL, Alert{})
	if err == nil || !strings.Contains(err.Error(), "blocked by network policy") {
		t.Fatalf("Deliver() error = %v, want blocked policy error", err)
	}
}

func TestEmailMessageSanitizesSubjectHeaders(t *testing.T) {
	t.Parallel()

	sentAt := time.Date(2026, 7, 24, 12, 30, 0, 0, time.UTC)
	message := emailMessage(
		&mail.Address{Name: "NetScope", Address: "alerts@example.test"},
		&mail.Address{Address: "ops@example.test"},
		Alert{
			Target:  Target{Name: "API\r\nBcc: attacker@example.test", Address: "api.test"},
			Status:  StatusUnavailable,
			Message: "check failed",
			SentAt:  sentAt,
		},
	)
	headers := strings.SplitN(message, "\r\n\r\n", 2)[0]
	if strings.Contains(headers, "\r\nBcc:") {
		t.Fatalf("message contains injected header:\n%s", headers)
	}
	if !strings.Contains(headers, "Subject: [NetScope] API  Bcc:") {
		t.Fatalf("sanitized subject missing:\n%s", headers)
	}
}
