package httpapi

import (
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/google/uuid"
)

func TestRunCSVExportsOneSafeRowPerResult(t *testing.T) {
	t.Parallel()

	runID := uuid.New()
	started := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	completed := started.Add(150 * time.Millisecond)
	run := diagnostics.DiagnosticRun{
		ID:              runID,
		TargetInput:     "=cmd|' /C calc'!A0",
		NormalizedHost:  "example.com",
		NormalizedURL:   "https://example.com",
		Status:          diagnostics.RunCompleted,
		RequestedChecks: []diagnostics.CheckType{diagnostics.CheckDNS},
		Options: diagnostics.RunOptions{
			TimeoutMS: 5000, HTTPMethod: "GET", FollowRedirects: true,
			MaxRedirects: 5, IPVersion: "auto", PingPackets: 4, MaxHops: 20,
		},
		Results: []diagnostics.CheckResult{{
			ID: uuid.New(), RunID: runID, Type: diagnostics.CheckDNS,
			Status: diagnostics.CheckPassed, DurationMS: 12,
			Summary: "Resolved host", Data: json.RawMessage(`{"a":["192.0.2.1"]}`),
			StartedAt: started, CompletedAt: completed,
		}},
		CreatedAt: started, StartedAt: &started, CompletedAt: &completed,
	}

	payload, err := runCSV(run)
	if err != nil {
		t.Fatalf("runCSV() error = %v", err)
	}
	records, err := csv.NewReader(strings.NewReader(string(payload))).ReadAll()
	if err != nil {
		t.Fatalf("parse CSV: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("rows = %d, want header and one result", len(records))
	}
	if records[1][0] != runID.String() || records[1][10] != "dns" {
		t.Fatalf("result row = %#v", records[1])
	}
	if records[1][1] != "'=cmd|' /C calc'!A0" {
		t.Fatalf("target cell was not formula-safe: %q", records[1][1])
	}
	if records[1][16] != `{"a":["192.0.2.1"]}` {
		t.Fatalf("data JSON = %q", records[1][16])
	}
}

func TestRunCSVIncludesRunWithoutResults(t *testing.T) {
	t.Parallel()

	payload, err := runCSV(diagnostics.DiagnosticRun{
		ID: uuid.New(), Status: diagnostics.RunQueued, CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("runCSV() error = %v", err)
	}
	records, err := csv.NewReader(strings.NewReader(string(payload))).ReadAll()
	if err != nil {
		t.Fatalf("parse CSV: %v", err)
	}
	if len(records) != 2 || records[1][10] != "" {
		t.Fatalf("records = %#v, want an empty check row", records)
	}
}
