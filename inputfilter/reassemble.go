package inputfilter

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// The bounds a Reassembler applies where its config leaves them
// (docs/decisions/0009-PLAN-streaming-content-engine.md Step A1.3): the
// window is the agent's, long against the 60 and 200 ms gaps measured and
// short against typing; the cap is the probe's
// (termcap/termcaptea/prober.go); and an X10 report's bytes come in one
// read, so the keys it ran into follow it within microseconds.
const (
	defaultWindow = 500 * time.Millisecond
	maxSequence   = 1024
	x10Window     = 5 * time.Millisecond
)

// ReplyMatcher tells a terminal's reply to a probe, which a Reassembler
// drops: a reply that comes late is not the user's typing. termcap's
// ParseReply tells one; inputfilter does not import termcap, so the
// program gives the matcher.
type ReplyMatcher interface {
	IsReply(raw string) bool
}

// ReplyMatcherFunc is a function used as a ReplyMatcher.
type ReplyMatcherFunc func(raw string) bool

// IsReply calls f.
func (f ReplyMatcherFunc) IsReply(raw string) bool { return f(raw) }

// ReassembleConfig is how a Reassembler joins. Its zero value takes the
// defaults.
type ReassembleConfig struct {
	// Window is how long a head waits for each next byte: 500 ms when it
	// is zero or less.
	Window time.Duration
	// Replies tells the replies to drop: none when it is nil.
	Replies ReplyMatcher
	// NoX10 leaves X10 reports as they came.
	NoX10 bool
}

// Kinds of sequence a Reassembler joins.
const (
	seqCSI    = 'c' // ends with a final byte, 0x40 to 0x7e
	seqString = 's' // OSC, DCS or APC: ends with BEL or ESC \
	seqX10    = 'x' // CSI M and three bytes
)

// Reassembler joins a report or a reply that a terminal's reader gave up
// on at its escape timeout, 50 ms in Bubble Tea's (docs/decisions/0009-MADR-streaming-content-engine.md
// §6, A1; docs/decisions/0009-PLAN-streaming-content-engine.md D14). It
// begins only from an unknown head, a termkey.InputUnknown that starts a
// CSI, OSC, DCS or APC sequence and does not end it, so that a bare Esc
// never waits. Each key after it that is the sequence's next byte is held,
// up to 1 KiB, until the sequence ends, a key that is not its next byte
// comes, or the window passes. A sequence joined is a reply the matcher
// knows, and dropped; or a Report, given as one InputUnknown with its
// whole Raw; or neither, and given back as it came. An X10 report that
// ConPTY re-encoded, CSI M and three runes, one of U+0080 to U+00FF, with
// the keys it ran into, is repaired, each rune its byte again. Anything
// else ends what is held first. A whole InputUnknown the matcher knows is
// dropped too. A Reassembler reads no clock, and is used on one
// goroutine.
type Reassembler struct {
	window  time.Duration
	replies ReplyMatcher
	x10     bool

	held  []termkey.Input // the head and the keys taken
	seq   []byte          // the sequence's bytes so far
	kind  byte            // seqCSI, seqString or seqX10; 0 with nothing held
	last  time.Time       // when the last piece came
	ready []termkey.Input // given by Tick, in order
}

// NewReassembler is a Reassembler with c's window and matcher.
func NewReassembler(c ReassembleConfig) *Reassembler {
	r := &Reassembler{window: defaultWindow, replies: c.Replies, x10: !c.NoX10}
	if c.Window > 0 {
		r.window = c.Window
	}
	return r
}

// Key gives the Reassembler in at now, and reports what it did; on
// Release, Tick gives what is ready.
func (r *Reassembler) Key(in termkey.Input, now time.Time) Action {
	r.expire(now)
	in.Raw = append([]byte(nil), in.Raw...) // held past the caller's buffer
	if r.kind != 0 {
		if b, ok := r.next(in); ok {
			r.held = append(r.held, in)
			r.seq = append(r.seq, b)
			r.last = now
			r.settle()
			return r.taken()
		}
		r.giveBack()
	}
	if r.head(in) {
		r.held = append(r.held[:0], in)
		r.last = now
		r.settle()
		return r.taken()
	}
	if in.Kind == termkey.InputUnknown && r.isReply(string(in.Raw)) {
		return r.taken()
	}
	if len(r.ready) == 0 {
		return Pass
	}
	r.ready = append(r.ready, in)
	return Release
}

// taken is the action for an input the Reassembler took.
func (r *Reassembler) taken() Action {
	if len(r.ready) > 0 {
		return Release
	}
	return Hold
}

// Tick gives the next input ready at now, oldest first: what Key released,
// and a head whose window has passed, given back as it came.
func (r *Reassembler) Tick(now time.Time) (termkey.Input, bool) {
	r.expire(now)
	if len(r.ready) == 0 {
		return termkey.Input{}, false
	}
	in := r.ready[0]
	r.ready = r.ready[1:]
	if len(r.ready) == 0 {
		r.ready = nil
	}
	return in, true
}

// Due is when Tick next has an input: the zero time when one is ready
// already, and false when nothing is held.
func (r *Reassembler) Due() (time.Time, bool) {
	switch {
	case len(r.ready) > 0:
		return time.Time{}, true
	case r.kind != 0:
		return r.last.Add(r.wait()), true
	}
	return time.Time{}, false
}

// wait is how long what is held waits for its next byte.
func (r *Reassembler) wait() time.Duration {
	if r.kind == seqX10 {
		return x10Window
	}
	return r.window
}

// expire gives back what is held when now has passed its wait.
func (r *Reassembler) expire(now time.Time) {
	if r.kind != 0 && now.Sub(r.last) >= r.wait() {
		r.giveBack()
	}
}

// drop forgets what is held and ready.
func (r *Reassembler) drop() {
	r.held, r.seq, r.kind, r.ready = r.held[:0], r.seq[:0], 0, nil
}

// head reports whether in begins a sequence to join, and sets the
// sequence's kind and bytes when it does.
func (r *Reassembler) head(in termkey.Input) bool {
	raw := in.Raw
	if in.Kind != termkey.InputUnknown || len(raw) < 3 || raw[0] != 0x1b {
		return false
	}
	if r.x10 && strings.HasPrefix(string(raw), "\x1b[M") {
		seq, ok := x10Bytes(raw[3:])
		if !ok {
			return false
		}
		r.kind, r.seq = seqX10, append(append(r.seq[:0], "\x1b[M"...), seq...)
		return true
	}
	switch raw[1] {
	case '[':
		for _, c := range raw[2:] {
			if c < 0x20 || c > 0x3f {
				return false // ended, or not a CSI
			}
		}
		r.kind = seqCSI
	case ']', 'P', '_':
		if raw[len(raw)-1] == 0x07 || strings.HasSuffix(string(raw), "\x1b\\") {
			return false
		}
		r.kind = seqString
	default:
		return false
	}
	r.seq = append(r.seq[:0], raw...)
	return true
}

// x10Bytes is the bytes of an X10 report's values, each rune ConPTY made
// of a byte of 0x80 or more its byte again, when one was re-encoded; ok is
// false for a report that needs no repair, or cannot have one.
func x10Bytes(rest []byte) ([]byte, bool) {
	var out []byte
	repaired := false
	for len(rest) > 0 {
		c, n := utf8.DecodeRune(rest)
		if c == utf8.RuneError && n == 1 || c > 0xff {
			return nil, false
		}
		repaired = repaired || c >= 0x80
		out = append(out, byte(c)) //nolint:gosec // c is U+00FF or under, checked above
		rest = rest[n:]
	}
	return out, repaired && len(out) <= 3
}

// next is the byte in adds to the sequence held, if it is its next one: a
// key typing one byte, ESC's or BEL's as a string sequence ends, or for an
// X10 report a key typing one rune of U+00FF or under. Key has given back
// already what waited past its window.
func (r *Reassembler) next(in termkey.Input) (byte, bool) {
	if in.Kind != termkey.InputKey || in.Key.Release {
		return 0, false
	}
	k := in.Key
	switch r.kind {
	case seqX10:
		c, n := utf8.DecodeRuneInString(k.Text)
		if n == 0 || n != len(k.Text) || c > 0xff {
			return 0, false
		}
		return byte(c), true //nolint:gosec // c is U+00FF or under, checked above
	case seqString:
		switch {
		case k.Code == 'g' && k.Mod&^locks == termkey.ModCtrl:
			return 0x07, true
		case k.Code == '\\' && r.seq[len(r.seq)-1] == 0x1b:
			return '\\', true
		}
	}
	if len(k.Text) == 1 && k.Text[0] >= 0x20 && k.Text[0] < 0x7f {
		return k.Text[0], true
	}
	return 0, false
}

// settle finishes the sequence held when it has ended, and gives it back
// when it has grown past the cap.
func (r *Reassembler) settle() {
	switch {
	case r.ended():
		raw := string(r.seq)
		switch {
		case r.isReply(raw):
		case r.isReport(raw):
			r.ready = append(r.ready, termkey.Input{Kind: termkey.InputUnknown, Raw: append([]byte(nil), r.seq...)})
		default:
			r.ready = append(r.ready, r.held...)
		}
		r.held, r.seq, r.kind = r.held[:0], r.seq[:0], 0
	case len(r.seq) > maxSequence:
		r.giveBack()
	}
}

// ended reports whether the sequence held is whole.
func (r *Reassembler) ended() bool {
	last := r.seq[len(r.seq)-1]
	switch r.kind {
	case seqX10:
		return len(r.seq) == 6
	case seqCSI:
		return last >= 0x40 && last <= 0x7e
	}
	return last == 0x07 || strings.HasSuffix(string(r.seq), "\x1b\\")
}

// isReply reports whether the matcher knows raw.
func (r *Reassembler) isReply(raw string) bool { return r.replies != nil && r.replies.IsReply(raw) }

// isReport reports whether raw is a Report.
func (r *Reassembler) isReport(raw string) bool {
	_, ok := ParseReport(raw)
	return ok
}

// giveBack readies what is held as it came.
func (r *Reassembler) giveBack() {
	r.ready = append(r.ready, r.held...)
	r.held, r.seq, r.kind = r.held[:0], r.seq[:0], 0
}
