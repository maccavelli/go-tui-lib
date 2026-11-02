package frame

import (
	"time"

	"github.com/maccavelli/go-tui-lib/v2/internal/enum"
)

// Demand is how often a view needs ticks to animate: None, Slow or Fast
// (docs/reports/0003-REPORT-agent-tui-ecosystem-research.md §8.17, grok's
// tick demand). Views drawn together tick at their Combine.
type Demand uint8

const (
	// None needs no tick: nothing animates. The zero value.
	None Demand = iota
	// Slow ticks a slow animation, such as a pulsing marker.
	Slow
	// Fast ticks every frame, as a spinner does.
	Fast
)

var demandNames = enum.Names[Demand]{
	Pkg: pkgName, Type: "Demand",
	Tokens: []string{"none", "slow", "fast"},
}

// String is the demand's token.
func (d Demand) String() string { return demandNames.String(d) }

// MarshalText is the demand's token. A demand with no token is an error.
func (d Demand) MarshalText() ([]byte, error) { return demandNames.Marshal(d) }

// UnmarshalText reads a token MarshalText wrote, exactly, case included.
func (d *Demand) UnmarshalText(b []byte) error { return demandNames.Unmarshal(b, d) }

// Combine is the greatest of d, the demand of views drawn together, and
// None for none.
func Combine(d ...Demand) Demand {
	var c Demand
	for _, v := range d {
		c = max(c, v)
	}
	return c
}

// defaultSlow is how often Slow ticks by default, grok's.
const defaultSlow = 83 * time.Millisecond

// Ticks is how often each Demand ticks. A zero or negative field takes its
// default.
type Ticks struct {
	Slow time.Duration // 83 ms by default
	Fast time.Duration // one frame at 60 fps by default
}

// Interval is how often d ticks: 0, no tick, for None and for a Demand
// with no token.
func (t Ticks) Interval(d Demand) time.Duration {
	switch d {
	case Slow:
		if t.Slow > 0 {
			return t.Slow
		}
		return defaultSlow
	case Fast:
		if t.Fast > 0 {
			return t.Fast
		}
		return time.Second / defaultFPS
	}
	return 0
}

// TickMsg is an animation's tick: when it fired, and the demand it was
// armed for.
type TickMsg struct {
	At     time.Time
	Demand Demand
}
