package diagnostics

import (
	"encoding/json"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/target"
	"github.com/google/uuid"
)

type CheckType string

const (
	CheckDNS        CheckType = "dns"
	CheckTCP        CheckType = "tcp"
	CheckHTTP       CheckType = "http"
	CheckTLS        CheckType = "tls"
	CheckPing       CheckType = "ping"
	CheckTraceroute CheckType = "traceroute"
)

type RunStatus string

const (
	RunQueued      RunStatus = "queued"
	RunRunning     RunStatus = "running"
	RunCompleted   RunStatus = "completed"
	RunPartial     RunStatus = "partial"
	RunFailed      RunStatus = "failed"
	RunCancelled   RunStatus = "cancelled"
	RunInterrupted RunStatus = "interrupted"
)

type CheckStatus string

const (
	CheckPending   CheckStatus = "pending"
	CheckRunning   CheckStatus = "running"
	CheckPassed    CheckStatus = "passed"
	CheckWarning   CheckStatus = "warning"
	CheckFailed    CheckStatus = "failed"
	CheckSkipped   CheckStatus = "skipped"
	CheckCancelled CheckStatus = "cancelled"
)

type RunOptions struct {
	TimeoutMS       int    `json:"timeoutMs"`
	TCPPorts        []int  `json:"tcpPorts,omitempty"`
	HTTPMethod      string `json:"httpMethod"`
	FollowRedirects bool   `json:"followRedirects"`
	MaxRedirects    int    `json:"maxRedirects"`
	IPVersion       string `json:"ipVersion"`
	PingPackets     int    `json:"pingPackets"`
	MaxHops         int    `json:"maxHops"`
}

type DiagnosticRun struct {
	ID              uuid.UUID       `json:"id"`
	TargetInput     string          `json:"target"`
	NormalizedHost  string          `json:"normalizedHost"`
	NormalizedURL   string          `json:"normalizedUrl,omitempty"`
	Target          target.Target   `json:"-"`
	Status          RunStatus       `json:"status"`
	RequestedChecks []CheckType     `json:"checks"`
	Options         RunOptions      `json:"options"`
	Summary         json.RawMessage `json:"summary,omitempty"`
	Results         []CheckResult   `json:"results"`
	CreatedAt       time.Time       `json:"createdAt"`
	StartedAt       *time.Time      `json:"startedAt,omitempty"`
	CompletedAt     *time.Time      `json:"completedAt,omitempty"`
	CancelledAt     *time.Time      `json:"cancelledAt,omitempty"`
}

type CheckResult struct {
	ID           uuid.UUID       `json:"id"`
	RunID        uuid.UUID       `json:"runId"`
	Type         CheckType       `json:"type"`
	Status       CheckStatus     `json:"status"`
	DurationMS   int64           `json:"durationMs"`
	Summary      string          `json:"summary,omitempty"`
	Data         json.RawMessage `json:"data"`
	ErrorCode    string          `json:"errorCode,omitempty"`
	ErrorMessage string          `json:"errorMessage,omitempty"`
	StartedAt    time.Time       `json:"startedAt"`
	CompletedAt  time.Time       `json:"completedAt"`
}

type ListFilter struct {
	Page     int
	PageSize int
	Status   RunStatus
}

type Page struct {
	Items      []DiagnosticRun `json:"items"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	TotalItems int64           `json:"totalItems"`
	TotalPages int             `json:"totalPages"`
}

func CalculateRunStatus(results []CheckResult, cancelled bool) RunStatus {
	if cancelled {
		return RunCancelled
	}
	if len(results) == 0 {
		return RunFailed
	}

	successes := 0
	failures := 0
	for _, result := range results {
		switch result.Status {
		case CheckPassed, CheckWarning:
			successes++
		case CheckFailed:
			failures++
		case CheckCancelled:
			return RunCancelled
		}
	}

	switch {
	case failures == 0 && successes == len(results):
		return RunCompleted
	case failures > 0 && successes > 0:
		return RunPartial
	default:
		return RunFailed
	}
}
