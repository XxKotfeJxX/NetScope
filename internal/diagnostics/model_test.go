package diagnostics

import "testing"

func TestCalculateRunStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		results   []CheckResult
		cancelled bool
		want      RunStatus
	}{
		{name: "completed", results: []CheckResult{{Status: CheckPassed}, {Status: CheckWarning}}, want: RunCompleted},
		{name: "partial", results: []CheckResult{{Status: CheckPassed}, {Status: CheckFailed}}, want: RunPartial},
		{name: "failed", results: []CheckResult{{Status: CheckFailed}}, want: RunFailed},
		{name: "cancelled result", results: []CheckResult{{Status: CheckCancelled}}, want: RunCancelled},
		{name: "cancelled context", results: []CheckResult{{Status: CheckPassed}}, cancelled: true, want: RunCancelled},
		{name: "empty", want: RunFailed},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := CalculateRunStatus(test.results, test.cancelled); got != test.want {
				t.Fatalf("CalculateRunStatus() = %q, want %q", got, test.want)
			}
		})
	}
}
