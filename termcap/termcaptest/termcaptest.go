// Package termcaptest runs a Bubble Tea program against a scripted fake
// terminal, so a program that embeds a termcap.Prober can be tested
// without a real one (docs/decisions/0005-MADR-terminal-capabilities-and-services.md
// Confirmation).
//
// A Terminal sits between a tea.Program's output and its input. It reads
// every escape sequence the program writes, and answers the queries its
// Profile answers, in the order they were asked, as a terminal does. It
// never touches a real terminal.
package termcaptest

import (
	"fmt"
	"image/color"
	"io"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/maccavelli/go-tui-lib/termcap"
)

// Profile is how a fake terminal answers. A zero field answers nothing.
type Profile struct {
	Name         string
	Env          []string             // the program's environment
	ColorProfile colorprofile.Profile // tea.WithColorProfile

	DA1           []int                    // the DA1 reply; nil answers nothing
	Modes         map[int]ansi.ModeSetting // DECRQM replies; nil answers none, a missing mode is not recognised
	KittyKeyboard bool                     // answers CSI ? u with the flags pushed
	Dark, Light   bool                     // answers DSR 996 with DSR 997
	Background    color.Color              // answers OSC 11
	Version       string                   // answers XTVERSION
	Notifications bool                     // answers the OSC 99 p=? query
	Graphics      bool                     // answers the Kitty graphics query with OK

	// Outer is the terminal a multiplexer runs in. Passthrough says whether
	// a tmux passthrough sequence reaches it (tmux's allow-passthrough).
	Outer       *Profile
	Passthrough bool
}

// xtermEnv is the environment of a terminal that names itself only by TERM.
const xtermEnv = "TERM=xterm-256color"

// Kitty is a terminal that answers every query.
func Kitty() Profile {
	return Profile{
		Name:          "kitty",
		Env:           []string{"TERM=xterm-kitty"},
		ColorProfile:  colorprofile.TrueColor,
		DA1:           []int{62, 22},
		Modes:         map[int]ansi.ModeSetting{1004: ansi.ModeReset, 2026: ansi.ModeReset, 2031: ansi.ModeReset, 2048: ansi.ModeReset},
		KittyKeyboard: true,
		Dark:          true,
		Background:    color.RGBA{R: 0x1e, G: 0x1e, B: 0x2e, A: 0xff},
		Version:       "kitty(0.39.1)",
		Notifications: true,
		Graphics:      true,
	}
}

// XTerm answers DA1, DECRQM and OSC 11, and none of the newer queries.
func XTerm() Profile {
	return Profile{
		Name:         "xterm",
		Env:          []string{xtermEnv},
		ColorProfile: colorprofile.ANSI256,
		DA1:          []int{65, 1, 9},
		Modes:        map[int]ansi.ModeSetting{1004: ansi.ModeReset},
		Background:   color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
	}
}

// Tmux is tmux inside a Kitty-like terminal. tmux answers DA1, XTVERSION,
// DECRQM 1004 and OSC 11 itself; a passthrough query reaches the outer
// terminal only when passthrough is allowed.
func Tmux(passthrough bool) Profile {
	outer := Kitty()
	name := "tmux"
	if !passthrough {
		name = "tmux without passthrough"
	}
	return Profile{
		Name:         name,
		Env:          []string{"TERM=tmux-256color", "TERM_PROGRAM=tmux", "TMUX=/tmp/tmux-1000/default,1,0"},
		ColorProfile: colorprofile.TrueColor,
		DA1:          []int{1, 2},
		Modes:        map[int]ansi.ModeSetting{1004: ansi.ModeReset},
		Background:   outer.Background,
		Version:      "tmux 3.4",
		Outer:        &outer,
		Passthrough:  passthrough,
	}
}

// DA1Only answers DA1 and nothing else.
func DA1Only() Profile {
	return Profile{Name: "DA1 only", Env: []string{xtermEnv}, ColorProfile: colorprofile.ANSI, DA1: []int{1}}
}

// Silent answers nothing, so a probe ends by its timeout.
func Silent() Profile {
	return Profile{Name: "silent", Env: []string{xtermEnv}, ColorProfile: colorprofile.ANSI}
}

// StopMsg is sent to the program once it has received its termcap.CapsMsg.
// The model must quit on it, through its prober's Quit, so that the
// terminal sees every mode restored.
type StopMsg struct{}

// RunTimeout bounds each wait in Run: for the CapsMsg, and for the program
// to quit after StopMsg.
var RunTimeout = 10 * time.Second

// Terminal is a scripted fake terminal speaking a Profile.
type Terminal struct {
	profile Profile

	mu      sync.Mutex
	out     strings.Builder // everything the program wrote
	pending string          // the start of a sequence not yet complete
	seqs    []string        // every escape sequence, in order
	flags   []int           // the Kitty keyboard flag stack
	input   chan string     // replies, in order
}

// NewTerminal returns a fake terminal that answers as p does.
func NewTerminal(p Profile) *Terminal {
	return &Terminal{profile: p, input: make(chan string, 1024)}
}

// Run runs model with a fake terminal speaking p. See Terminal.Run.
func Run(tb testing.TB, model tea.Model, p Profile) termcap.Caps {
	tb.Helper()
	return NewTerminal(p).Run(tb, model)
}

// Run runs model as a tea.Program against the terminal until it has
// received a termcap.CapsMsg, sends it StopMsg, waits for it to quit, and
// returns that CapsMsg's Caps. It fails tb when either wait passes
// RunTimeout, or the program fails.
func (t *Terminal) Run(tb testing.TB, model tea.Model) termcap.Caps {
	tb.Helper()
	pr, pw := io.Pipe()
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case s := <-t.input:
				if _, err := io.WriteString(pw, s); err != nil {
					return
				}
			case <-stop:
				return
			}
		}
	}()
	defer func() {
		close(stop)
		for _, c := range []io.Closer{pr, pw} {
			if err := c.Close(); err != nil {
				tb.Errorf("%s: closing the input pipe: %v", t.profile.Name, err)
			}
		}
	}()

	w := &watcher{inner: model, caps: make(chan termcap.Caps, 1)}
	prog := tea.NewProgram(w,
		tea.WithInput(pr),
		tea.WithOutput(t),
		tea.WithEnvironment(t.profile.Env),
		tea.WithColorProfile(t.profile.ColorProfile),
		tea.WithWindowSize(80, 24),
		tea.WithoutSignalHandler(),
	)
	done := make(chan error, 1)
	go func() {
		_, err := prog.Run()
		done <- err
	}()

	var caps termcap.Caps
	select {
	case caps = <-w.caps:
	case err := <-done:
		tb.Fatalf("%s: the program exited before its CapsMsg: %v", t.profile.Name, err)
	case <-time.After(RunTimeout):
		prog.Kill()
		<-done
		tb.Fatalf("%s: no CapsMsg within %v", t.profile.Name, RunTimeout)
	}
	prog.Send(StopMsg{})
	select {
	case err := <-done:
		if err != nil {
			tb.Fatalf("%s: %v", t.profile.Name, err)
		}
	case <-time.After(RunTimeout):
		prog.Kill()
		<-done
		tb.Fatalf("%s: the program did not quit within %v of StopMsg", t.profile.Name, RunTimeout)
	}
	return caps
}

// Send writes seq to the program's input, as the terminal would, after
// every reply already queued.
func (t *Terminal) Send(seq string) { t.input <- seq }

// Output is everything the program wrote.
func (t *Terminal) Output() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.out.String()
}

// Sequences is every escape sequence the program wrote, in order.
func (t *Terminal) Sequences() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.seqs...)
}

// Write takes the program's output, and answers each complete query in it.
func (t *Terminal) Write(b []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.out.Write(b)
	s := t.pending + string(b)
	for {
		i := strings.IndexByte(s, ansi.ESC)
		if i < 0 {
			s = ""
			break
		}
		n := seqLen(s[i:])
		if n == 0 {
			s = s[i:]
			break
		}
		seq := s[i : i+n]
		s = s[i+n:]
		t.seqs = append(t.seqs, seq)
		if r := t.answer(&t.profile, seq); r != "" {
			t.input <- r
		}
	}
	t.pending = s
	return len(b), nil
}

// seqLen is the length of the escape sequence s starts with, or 0 when it
// is not complete yet.
func seqLen(s string) int {
	if len(s) < 2 {
		return 0
	}
	switch s[1] {
	case '[': // CSI: parameters, then a final byte
		for i := 2; i < len(s); i++ {
			if s[i] >= 0x40 && s[i] <= 0x7e {
				return i + 1
			}
		}
		return 0
	case ']': // OSC: ends with BEL or ST
		for i := 2; i < len(s); i++ {
			if s[i] == ansi.BEL {
				return i + 1
			}
			if s[i] == ansi.ESC && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
		}
		return 0
	case 'P', '_', '^', 'X': // DCS, APC, PM, SOS: end with ST; ESC ESC is an escaped ESC
		for i := 2; i < len(s)-1; i++ {
			if s[i] == ansi.ESC {
				if s[i+1] == '\\' {
					return i + 2
				}
				i++
			}
		}
		return 0
	}
	return 2
}

// answer is p's reply to seq, or "".
func (t *Terminal) answer(p *Profile, seq string) string {
	switch {
	case seq == ansi.RequestPrimaryDeviceAttributes:
		if p.DA1 == nil {
			return ""
		}
		return "\x1b[?" + joinInts(p.DA1) + "c"

	case strings.HasPrefix(seq, "\x1b[?") && strings.HasSuffix(seq, "$p"):
		mode, err := strconv.Atoi(seq[3 : len(seq)-2])
		if err != nil || p.Modes == nil {
			return ""
		}
		return fmt.Sprintf("\x1b[?%d;%d$y", mode, p.Modes[mode])

	case seq == ansi.RequestKittyKeyboard:
		if !p.KittyKeyboard {
			return ""
		}
		flags := 0
		if n := len(t.flags); n > 0 {
			flags = t.flags[n-1]
		}
		return fmt.Sprintf("\x1b[?%du", flags)
	case strings.HasPrefix(seq, "\x1b[>") && strings.HasSuffix(seq, "u"):
		if f, err := strconv.Atoi(seq[3 : len(seq)-1]); err == nil && p.KittyKeyboard {
			t.flags = append(t.flags, f)
		}
		return ""
	case strings.HasPrefix(seq, "\x1b[<") && strings.HasSuffix(seq, "u"):
		if n := len(t.flags); n > 0 {
			t.flags = t.flags[:n-1]
		}
		return ""

	case seq == ansi.RequestLightDarkReport:
		switch {
		case p.Dark:
			return "\x1b[?997;1n"
		case p.Light:
			return "\x1b[?997;2n"
		}
		return ""

	case seq == ansi.RequestNameVersion:
		if p.Version == "" {
			return ""
		}
		return "\x1bP>|" + p.Version + "\x1b\\"

	case strings.HasPrefix(seq, "\x1b]11;?"):
		if p.Background == nil {
			return ""
		}
		r, g, b, _ := p.Background.RGBA()
		return fmt.Sprintf("\x1b]11;rgb:%04x/%04x/%04x\x1b\\", r, g, b)

	case strings.HasPrefix(seq, "\x1b]99;"):
		meta, _, _ := strings.Cut(strings.TrimPrefix(seq, "\x1b]99;"), ";")
		if !p.Notifications || !strings.Contains(meta, "p=?") {
			return ""
		}
		return "\x1b]99;" + meta + ";a=focus,report:o=always,unfocused,invisible:u=0,1,2\x1b\\"

	case strings.HasPrefix(seq, "\x1b_G"):
		opts, _, _ := strings.Cut(strings.TrimPrefix(seq, "\x1b_G"), ";")
		if !p.Graphics || !strings.Contains(","+opts+",", ",a=q,") {
			return ""
		}
		id := "0"
		for o := range strings.SplitSeq(opts, ",") {
			if v, ok := strings.CutPrefix(o, "i="); ok {
				id = v
			}
		}
		return "\x1b_Gi=" + id + ";OK\x1b\\"

	case strings.HasPrefix(seq, "\x1bPtmux;"):
		if p.Outer == nil || !p.Passthrough {
			return ""
		}
		inner := strings.ReplaceAll(strings.TrimSuffix(strings.TrimPrefix(seq, "\x1bPtmux;"), "\x1b\\"), "\x1b\x1b", "\x1b")
		return t.answer(p.Outer, inner)
	}
	return ""
}

func joinInts(v []int) string {
	s := make([]string, len(v))
	for i, n := range v {
		s[i] = strconv.Itoa(n)
	}
	return strings.Join(s, ";")
}

// watcher passes every message to the program's model, and catches the
// first CapsMsg on its way.
type watcher struct {
	inner tea.Model
	caps  chan termcap.Caps
}

func (w *watcher) Init() tea.Cmd { return w.inner.Init() }

func (w *watcher) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if c, ok := msg.(termcap.CapsMsg); ok {
		select {
		case w.caps <- c.Caps:
		default:
		}
	}
	m, cmd := w.inner.Update(msg)
	w.inner = m
	return w, cmd
}

func (w *watcher) View() tea.View { return w.inner.View() }
