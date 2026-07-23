package probe

import "github.com/XxKotfeJxX/netscope/internal/diagnostics"

// Probe is kept as a public alias at the probe package boundary while the
// diagnostics service owns the interface it consumes.
type Probe = diagnostics.Probe
