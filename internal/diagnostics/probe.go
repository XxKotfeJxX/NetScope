package diagnostics

import (
	"context"

	"github.com/XxKotfeJxX/netscope/internal/target"
)

type Probe interface {
	Type() CheckType
	Run(context.Context, target.Target, RunOptions) CheckResult
}
