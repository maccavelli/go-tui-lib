package termsvc

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/termcap"
)

// TitleRunes is the longest window title SanitizeTitle keeps.
const TitleRunes = 240

// bidi are the bidirectional controls, which can make a title read
// differently from what it holds.
var bidi = []rune{0x061c, 0x200e, 0x200f, 0x202a, 0x202b, 0x202c, 0x202d, 0x202e, 0x2066, 0x2067, 0x2068, 0x2069}

// SanitizeTitle is s fit for View.WindowTitle: escape sequences, controls
// and bidirectional marks removed, and at most TitleRunes characters. A
// program not on a terminal sets no title.
func SanitizeTitle(s string) string {
	s = strings.Map(func(r rune) rune {
		if slices.Contains(bidi, r) {
			return -1
		}
		return r
	}, strip(ansi.Strip(s)))
	if r := []rune(s); len(r) > TitleRunes {
		s = string(r[:TitleRunes])
	}
	return s
}

// ActivityState is what an agent is doing, as Kilo's activity beacon
// names it.
type ActivityState uint8

// Activity states.
const (
	ActivityIdle ActivityState = iota
	ActivityBusy
	ActivityRetry
	ActivityWaiting
	ActivityError
	ActivityDone
)

var activityNames = []string{"idle", "busy", "retry", "waiting", "error", "done"}

// String is the state's name in the beacon.
func (s ActivityState) String() string {
	if int(s) < len(activityNames) {
		return activityNames[s]
	}
	return strconv.Itoa(int(s))
}

// activityVersion is the beacon format's version.
const activityVersion = "1"

// Activity is the beacon sequence, OSC 777 ; vendor ; activity ; 1 ;
// state ; unix-ms, for a host such as an editor that embeds the terminal.
// The vendor tag keeps it apart from rxvt's own OSC 777. A program writes
// it on each change of state, and every ActivityInterval.
func Activity(vendor string, s ActivityState, t time.Time) string {
	return ansi.URxvtExt(strings.ReplaceAll(strip(vendor), ";", ""),
		"activity", activityVersion, s.String(), strconv.FormatInt(t.UnixMilli(), 10))
}

// ActivityReport is a beacon read back.
type ActivityReport struct {
	Vendor string
	State  ActivityState
	At     time.Time
}

// Beacon limits: a report from further ahead, or older, is refused, so a
// clock skew or a dead program never leaves a stale state.
const (
	ActivityAhead  = 5 * time.Second
	ActivityMaxAge = 15 * time.Second
)

// ErrActivity is ParseActivity's error.
var ErrActivity = errors.New("termsvc: not an activity beacon")

// ParseActivity reads a beacon's payload, the part after "777;":
// vendor;activity;version;state;unix-ms. It refuses another version, an
// unknown state, and a time more than ActivityAhead after now or
// ActivityMaxAge or more before it.
func ParseActivity(payload string, now time.Time) (ActivityReport, error) {
	f := strings.Split(payload, ";")
	if len(f) != 5 || f[1] != "activity" {
		return ActivityReport{}, ErrActivity
	}
	if f[2] != activityVersion {
		return ActivityReport{}, fmt.Errorf("%w: version %q", ErrActivity, f[2])
	}
	state, ok := parseActivityState(f[3])
	if !ok {
		return ActivityReport{}, fmt.Errorf("%w: state %q", ErrActivity, f[3])
	}
	ms, err := strconv.ParseInt(f[4], 10, 64)
	if err != nil {
		return ActivityReport{}, fmt.Errorf("%w: time %q", ErrActivity, f[4])
	}
	at := time.UnixMilli(ms)
	if at.Sub(now) > ActivityAhead || now.Sub(at) >= ActivityMaxAge {
		return ActivityReport{}, fmt.Errorf("%w: time %v from %v", ErrActivity, at, now)
	}
	return ActivityReport{Vendor: f[0], State: state, At: at}, nil
}

// parseActivityState reads a state's name.
func parseActivityState(name string) (ActivityState, bool) {
	for s := ActivityIdle; s <= ActivityDone; s++ {
		if s.String() == name {
			return s, true
		}
	}
	return ActivityIdle, false
}

// ActivityInterval is how often the beacon repeats.
const ActivityInterval = 5 * time.Second

// ActivityTickMsg is ActivityBeacon's tick.
type ActivityTickMsg struct{ At time.Time }

// ActivityBeacon is the beacon's timer: it delivers ActivityTickMsg after
// ActivityInterval. A program that opts in writes Activity on each tick
// and returns ActivityBeacon again.
func ActivityBeacon() tea.Cmd {
	return tea.Tick(ActivityInterval, func(t time.Time) tea.Msg { return ActivityTickMsg{At: t} })
}

// Pointer is the OSC 22 sequence that sets the mouse pointer's shape, such
// as "pointer" over a link, or "" for the terminal's default. Only Ghostty
// and kitty outside a multiplexer take it; elsewhere it is "".
func Pointer(c termcap.Caps, shape string) string {
	brand := c.Brand.Value
	if c.Mux.Value != termcap.NoMux || brand != termcap.BrandGhostty && brand != termcap.BrandKitty {
		return ""
	}
	shape = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r == '-' {
			return r
		}
		return -1
	}, shape)
	if shape == "" && brand == termcap.BrandGhostty {
		shape = "default" // kitty resets with an empty OSC 22
	}
	return "\x1b]22;" + shape + "\x07"
}

// ProgressSupported reports whether View.ProgressBar (OSC 9;4) shows as a
// progress bar: on Ghostty, WezTerm, and iTerm2 3.6 or later, whose
// version comes from its XTVERSION reply. Older iTerm2 shows it as alert
// text.
func ProgressSupported(c termcap.Caps) bool {
	switch c.Brand.Value {
	case termcap.BrandGhostty, termcap.BrandWezTerm:
		return true
	case termcap.BrandITerm2:
		v, ok := strings.CutPrefix(c.Terminal.Value, "iTerm2 ")
		return ok && versionAtLeast(v, 3, 6)
	}
	return false
}

// versionAtLeast reports whether the dotted version v is at least
// major.minor.
func versionAtLeast(v string, major, minor int) bool {
	ma, rest, _ := strings.Cut(v, ".")
	mi, _, _ := strings.Cut(rest, ".")
	m, err := strconv.Atoi(ma)
	if err != nil {
		return false
	}
	n, err := strconv.Atoi(mi)
	if err != nil {
		n = 0
	}
	return m > major || m == major && n >= minor
}
