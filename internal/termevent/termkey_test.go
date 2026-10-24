package termevent

import (
	"bufio"
	"context"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// keyCodes pairs each termkey key code with ultraviolet's of the same name
// (0020-PLAN-charm-free-cores-glyphs-and-safetext.md Step 11). It is
// generated from ultraviolet's key.go, in its order.
var keyCodes = []struct {
	name   string
	ours   rune
	theirs rune
}{
	{"KeyUp", termkey.KeyUp, uv.KeyUp},
	{"KeyDown", termkey.KeyDown, uv.KeyDown},
	{"KeyRight", termkey.KeyRight, uv.KeyRight},
	{"KeyLeft", termkey.KeyLeft, uv.KeyLeft},
	{"KeyBegin", termkey.KeyBegin, uv.KeyBegin},
	{"KeyFind", termkey.KeyFind, uv.KeyFind},
	{"KeyInsert", termkey.KeyInsert, uv.KeyInsert},
	{"KeyDelete", termkey.KeyDelete, uv.KeyDelete},
	{"KeySelect", termkey.KeySelect, uv.KeySelect},
	{"KeyPgUp", termkey.KeyPgUp, uv.KeyPgUp},
	{"KeyPgDown", termkey.KeyPgDown, uv.KeyPgDown},
	{"KeyHome", termkey.KeyHome, uv.KeyHome},
	{"KeyEnd", termkey.KeyEnd, uv.KeyEnd},
	{"KeyKpEnter", termkey.KeyKpEnter, uv.KeyKpEnter},
	{"KeyKpEqual", termkey.KeyKpEqual, uv.KeyKpEqual},
	{"KeyKpMultiply", termkey.KeyKpMultiply, uv.KeyKpMultiply},
	{"KeyKpPlus", termkey.KeyKpPlus, uv.KeyKpPlus},
	{"KeyKpComma", termkey.KeyKpComma, uv.KeyKpComma},
	{"KeyKpMinus", termkey.KeyKpMinus, uv.KeyKpMinus},
	{"KeyKpDecimal", termkey.KeyKpDecimal, uv.KeyKpDecimal},
	{"KeyKpDivide", termkey.KeyKpDivide, uv.KeyKpDivide},
	{"KeyKp0", termkey.KeyKp0, uv.KeyKp0},
	{"KeyKp1", termkey.KeyKp1, uv.KeyKp1},
	{"KeyKp2", termkey.KeyKp2, uv.KeyKp2},
	{"KeyKp3", termkey.KeyKp3, uv.KeyKp3},
	{"KeyKp4", termkey.KeyKp4, uv.KeyKp4},
	{"KeyKp5", termkey.KeyKp5, uv.KeyKp5},
	{"KeyKp6", termkey.KeyKp6, uv.KeyKp6},
	{"KeyKp7", termkey.KeyKp7, uv.KeyKp7},
	{"KeyKp8", termkey.KeyKp8, uv.KeyKp8},
	{"KeyKp9", termkey.KeyKp9, uv.KeyKp9},
	{"KeyKpSep", termkey.KeyKpSep, uv.KeyKpSep},
	{"KeyKpUp", termkey.KeyKpUp, uv.KeyKpUp},
	{"KeyKpDown", termkey.KeyKpDown, uv.KeyKpDown},
	{"KeyKpLeft", termkey.KeyKpLeft, uv.KeyKpLeft},
	{"KeyKpRight", termkey.KeyKpRight, uv.KeyKpRight},
	{"KeyKpPgUp", termkey.KeyKpPgUp, uv.KeyKpPgUp},
	{"KeyKpPgDown", termkey.KeyKpPgDown, uv.KeyKpPgDown},
	{"KeyKpHome", termkey.KeyKpHome, uv.KeyKpHome},
	{"KeyKpEnd", termkey.KeyKpEnd, uv.KeyKpEnd},
	{"KeyKpInsert", termkey.KeyKpInsert, uv.KeyKpInsert},
	{"KeyKpDelete", termkey.KeyKpDelete, uv.KeyKpDelete},
	{"KeyKpBegin", termkey.KeyKpBegin, uv.KeyKpBegin},
	{"KeyF1", termkey.KeyF1, uv.KeyF1},
	{"KeyF2", termkey.KeyF2, uv.KeyF2},
	{"KeyF3", termkey.KeyF3, uv.KeyF3},
	{"KeyF4", termkey.KeyF4, uv.KeyF4},
	{"KeyF5", termkey.KeyF5, uv.KeyF5},
	{"KeyF6", termkey.KeyF6, uv.KeyF6},
	{"KeyF7", termkey.KeyF7, uv.KeyF7},
	{"KeyF8", termkey.KeyF8, uv.KeyF8},
	{"KeyF9", termkey.KeyF9, uv.KeyF9},
	{"KeyF10", termkey.KeyF10, uv.KeyF10},
	{"KeyF11", termkey.KeyF11, uv.KeyF11},
	{"KeyF12", termkey.KeyF12, uv.KeyF12},
	{"KeyF13", termkey.KeyF13, uv.KeyF13},
	{"KeyF14", termkey.KeyF14, uv.KeyF14},
	{"KeyF15", termkey.KeyF15, uv.KeyF15},
	{"KeyF16", termkey.KeyF16, uv.KeyF16},
	{"KeyF17", termkey.KeyF17, uv.KeyF17},
	{"KeyF18", termkey.KeyF18, uv.KeyF18},
	{"KeyF19", termkey.KeyF19, uv.KeyF19},
	{"KeyF20", termkey.KeyF20, uv.KeyF20},
	{"KeyF21", termkey.KeyF21, uv.KeyF21},
	{"KeyF22", termkey.KeyF22, uv.KeyF22},
	{"KeyF23", termkey.KeyF23, uv.KeyF23},
	{"KeyF24", termkey.KeyF24, uv.KeyF24},
	{"KeyF25", termkey.KeyF25, uv.KeyF25},
	{"KeyF26", termkey.KeyF26, uv.KeyF26},
	{"KeyF27", termkey.KeyF27, uv.KeyF27},
	{"KeyF28", termkey.KeyF28, uv.KeyF28},
	{"KeyF29", termkey.KeyF29, uv.KeyF29},
	{"KeyF30", termkey.KeyF30, uv.KeyF30},
	{"KeyF31", termkey.KeyF31, uv.KeyF31},
	{"KeyF32", termkey.KeyF32, uv.KeyF32},
	{"KeyF33", termkey.KeyF33, uv.KeyF33},
	{"KeyF34", termkey.KeyF34, uv.KeyF34},
	{"KeyF35", termkey.KeyF35, uv.KeyF35},
	{"KeyF36", termkey.KeyF36, uv.KeyF36},
	{"KeyF37", termkey.KeyF37, uv.KeyF37},
	{"KeyF38", termkey.KeyF38, uv.KeyF38},
	{"KeyF39", termkey.KeyF39, uv.KeyF39},
	{"KeyF40", termkey.KeyF40, uv.KeyF40},
	{"KeyF41", termkey.KeyF41, uv.KeyF41},
	{"KeyF42", termkey.KeyF42, uv.KeyF42},
	{"KeyF43", termkey.KeyF43, uv.KeyF43},
	{"KeyF44", termkey.KeyF44, uv.KeyF44},
	{"KeyF45", termkey.KeyF45, uv.KeyF45},
	{"KeyF46", termkey.KeyF46, uv.KeyF46},
	{"KeyF47", termkey.KeyF47, uv.KeyF47},
	{"KeyF48", termkey.KeyF48, uv.KeyF48},
	{"KeyF49", termkey.KeyF49, uv.KeyF49},
	{"KeyF50", termkey.KeyF50, uv.KeyF50},
	{"KeyF51", termkey.KeyF51, uv.KeyF51},
	{"KeyF52", termkey.KeyF52, uv.KeyF52},
	{"KeyF53", termkey.KeyF53, uv.KeyF53},
	{"KeyF54", termkey.KeyF54, uv.KeyF54},
	{"KeyF55", termkey.KeyF55, uv.KeyF55},
	{"KeyF56", termkey.KeyF56, uv.KeyF56},
	{"KeyF57", termkey.KeyF57, uv.KeyF57},
	{"KeyF58", termkey.KeyF58, uv.KeyF58},
	{"KeyF59", termkey.KeyF59, uv.KeyF59},
	{"KeyF60", termkey.KeyF60, uv.KeyF60},
	{"KeyF61", termkey.KeyF61, uv.KeyF61},
	{"KeyF62", termkey.KeyF62, uv.KeyF62},
	{"KeyF63", termkey.KeyF63, uv.KeyF63},
	{"KeyCapsLock", termkey.KeyCapsLock, uv.KeyCapsLock},
	{"KeyScrollLock", termkey.KeyScrollLock, uv.KeyScrollLock},
	{"KeyNumLock", termkey.KeyNumLock, uv.KeyNumLock},
	{"KeyPrintScreen", termkey.KeyPrintScreen, uv.KeyPrintScreen},
	{"KeyPause", termkey.KeyPause, uv.KeyPause},
	{"KeyMenu", termkey.KeyMenu, uv.KeyMenu},
	{"KeyMediaPlay", termkey.KeyMediaPlay, uv.KeyMediaPlay},
	{"KeyMediaPause", termkey.KeyMediaPause, uv.KeyMediaPause},
	{"KeyMediaPlayPause", termkey.KeyMediaPlayPause, uv.KeyMediaPlayPause},
	{"KeyMediaReverse", termkey.KeyMediaReverse, uv.KeyMediaReverse},
	{"KeyMediaStop", termkey.KeyMediaStop, uv.KeyMediaStop},
	{"KeyMediaFastForward", termkey.KeyMediaFastForward, uv.KeyMediaFastForward},
	{"KeyMediaRewind", termkey.KeyMediaRewind, uv.KeyMediaRewind},
	{"KeyMediaNext", termkey.KeyMediaNext, uv.KeyMediaNext},
	{"KeyMediaPrev", termkey.KeyMediaPrev, uv.KeyMediaPrev},
	{"KeyMediaRecord", termkey.KeyMediaRecord, uv.KeyMediaRecord},
	{"KeyLowerVol", termkey.KeyLowerVol, uv.KeyLowerVol},
	{"KeyRaiseVol", termkey.KeyRaiseVol, uv.KeyRaiseVol},
	{"KeyMute", termkey.KeyMute, uv.KeyMute},
	{"KeyLeftShift", termkey.KeyLeftShift, uv.KeyLeftShift},
	{"KeyLeftAlt", termkey.KeyLeftAlt, uv.KeyLeftAlt},
	{"KeyLeftCtrl", termkey.KeyLeftCtrl, uv.KeyLeftCtrl},
	{"KeyLeftSuper", termkey.KeyLeftSuper, uv.KeyLeftSuper},
	{"KeyLeftHyper", termkey.KeyLeftHyper, uv.KeyLeftHyper},
	{"KeyLeftMeta", termkey.KeyLeftMeta, uv.KeyLeftMeta},
	{"KeyRightShift", termkey.KeyRightShift, uv.KeyRightShift},
	{"KeyRightAlt", termkey.KeyRightAlt, uv.KeyRightAlt},
	{"KeyRightCtrl", termkey.KeyRightCtrl, uv.KeyRightCtrl},
	{"KeyRightSuper", termkey.KeyRightSuper, uv.KeyRightSuper},
	{"KeyRightHyper", termkey.KeyRightHyper, uv.KeyRightHyper},
	{"KeyRightMeta", termkey.KeyRightMeta, uv.KeyRightMeta},
	{"KeyIsoLevel3Shift", termkey.KeyIsoLevel3Shift, uv.KeyIsoLevel3Shift},
	{"KeyIsoLevel5Shift", termkey.KeyIsoLevel5Shift, uv.KeyIsoLevel5Shift},
	{"KeyBackspace", termkey.KeyBackspace, uv.KeyBackspace},
	{"KeyTab", termkey.KeyTab, uv.KeyTab},
	{"KeyEnter", termkey.KeyEnter, uv.KeyEnter},
	{"KeyEscape", termkey.KeyEscape, uv.KeyEscape},
	{"KeySpace", termkey.KeySpace, uv.KeySpace},
}

// modBits pairs each termkey modifier with ultraviolet's.
var modBits = []struct {
	name   string
	ours   termkey.Mod
	theirs uv.KeyMod
}{
	{"ModShift", termkey.ModShift, uv.ModShift},
	{"ModAlt", termkey.ModAlt, uv.ModAlt},
	{"ModCtrl", termkey.ModCtrl, uv.ModCtrl},
	{"ModMeta", termkey.ModMeta, uv.ModMeta},
	{"ModHyper", termkey.ModHyper, uv.ModHyper},
	{"ModSuper", termkey.ModSuper, uv.ModSuper},
	{"ModCapsLock", termkey.ModCapsLock, uv.ModCapsLock},
	{"ModNumLock", termkey.ModNumLock, uv.ModNumLock},
	{"ModScrollLock", termkey.ModScrollLock, uv.ModScrollLock},
}

// TestKeyCodesMatchUltraviolet holds MADR D7: each termkey code and
// modifier bit has ultraviolet's value, which Bubble Tea's tea.Key takes,
// and termkey names each as ultraviolet's Keystroke does.
func TestKeyCodesMatchUltraviolet(t *testing.T) {
	for _, c := range keyCodes {
		if c.ours != c.theirs {
			t.Errorf("%s: termkey %d, ultraviolet %d", c.name, c.ours, c.theirs)
		}
		ours, theirs := termkey.KeyEvent{Code: c.ours}.Keystroke(), uv.Key{Code: c.theirs}.Keystroke()
		if ours != theirs {
			t.Errorf("%s: termkey names it %q, ultraviolet %q", c.name, ours, theirs)
		}
	}
	for _, m := range modBits {
		if int(m.ours) != int(m.theirs) {
			t.Errorf("%s: termkey %d, ultraviolet %d", m.name, m.ours, m.theirs)
		}
		for _, code := range []rune{'a', termkey.KeyUp, termkey.KeyLeftShift, termkey.KeyRightSuper} {
			ours := termkey.KeyEvent{Code: code, Mod: m.ours}.Keystroke()
			theirs := uv.Key{Code: code, Mod: m.theirs}.Keystroke()
			if ours != theirs {
				t.Errorf("%s with code %d: termkey names it %q, ultraviolet %q", m.name, code, ours, theirs)
			}
		}
	}
	all := termkey.ModShift | termkey.ModAlt | termkey.ModCtrl | termkey.ModMeta | termkey.ModHyper | termkey.ModSuper
	if ours, theirs := (termkey.KeyEvent{Code: 'a', Mod: all}).Keystroke(), (uv.Key{Code: 'a', Mod: uv.KeyMod(all)}).Keystroke(); ours != theirs {
		t.Errorf("every modifier: termkey names it %q, ultraviolet %q", ours, theirs)
	}
	if termkey.KeyExtended != uv.KeyExtended {
		t.Errorf("KeyExtended: termkey %d, ultraviolet %d", termkey.KeyExtended, uv.KeyExtended)
	}
}

// differ is each corpus row the two decoders read differently, and why. A
// row here whose decoders agree fails as stale.
var differ = map[string]string{
	"the replacement character itself": "ultraviolet reads a well-formed U+FFFD as three bytes it does not decode; " +
		"termkey reads it as the character",
	"a grapheme of two runes": "termkey reads a key a rune, as it imports nothing for grapheme clusters; " +
		"ultraviolet reads the cluster as one KeyExtended key",
	"Kitty ctrl+ф, its base key": "ultraviolet's Kitty parser falls through from the base-layout key to the " +
		"shifted key, so ShiftedCode is the base-layout key; termkey keeps the shifted key the terminal sent",
	"Kitty shift+ф, both keys": "ultraviolet's fall-through sets ShiftedCode, and so Text, to the base-layout " +
		"key (\"a\" for shift+ф); termkey keeps the shifted key and its text",
	"a win32-input-mode key": "MADR D7 leaves win32-input-mode out of scope: termkey gives it as a " +
		"sequence it does not decode, where ultraviolet decodes the key",
	"a paste holding an OSC": "ultraviolet drops a sequence it does not decode from a paste; termkey keeps " +
		"the paste's bytes as they came",
	"a paste not ended": "ultraviolet holds a paste the input left unended; termkey's Flush gives it as " +
		"the paste so far",
	"a paste end with no start": "ultraviolet gives a paste end with no start as an empty paste; termkey " +
		"gives it as a sequence it does not decode",
	"OSC, cut by an ESC": "ultraviolet consumes the ESC that cuts a string; termkey leaves it to begin " +
		"what follows, so a cut reply never ends in the bare ESC termcap.ParseReply reads as an end",
	"33 parameters": "Bubble Tea's reader panics on it: x/ansi v0.11.8's parser runs past its " +
		"32-parameter buffer, which termcap.ParseReply guards against too; termkey gives the sequence " +
		"as one it does not decode",
}

// event is a key or a paste, as both decoders give it.
type event struct {
	paste bool
	text  string
	key   termkey.KeyEvent
}

// corpusRow is one line of termkey's corpus.
type corpusRow struct {
	name string
	in   string
}

// readCorpus reads termkey/testdata/golden/corpus.golden, which termkey's
// TestCorpus keeps: a width annotation, then a name and a quoted input.
func readCorpus(t *testing.T) []corpusRow {
	t.Helper()
	f, err := os.Open("../../termkey/testdata/golden/corpus.golden")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var rows []corpusRow
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		_, line, ok := strings.Cut(sc.Text(), "|")
		name, quoted, tab := strings.Cut(line, "\t")
		in, qerr := strconv.Unquote(quoted)
		if !ok || !tab || qerr != nil {
			t.Fatalf("corpus line %q is not a width, a name and a quoted input", sc.Text())
		}
		rows = append(rows, corpusRow{name, in})
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("the corpus is empty")
	}
	return rows
}

// ourEvents is what termkey's Decoder gives for in, read whole and
// flushed, as keys and pastes.
func ourEvents(in string) []event {
	var d termkey.Decoder
	var evs []event
	for _, i := range d.Flush(d.Decode([]byte(in), nil)) {
		switch i.Kind {
		case termkey.InputKey:
			evs = append(evs, event{key: i.Key})
		case termkey.InputPaste:
			evs = append(evs, event{paste: true, text: i.Text})
		}
	}
	return evs
}

// theirEvents is what Bubble Tea's reader, ultraviolet's TerminalReader,
// gives for in read whole and then the end of the input, which reads what
// it holds as the escape timeout would, as keys and pastes. A panic in the
// reader is returned as panicked.
func theirEvents(t *testing.T, in string) (evs []event, panicked any) {
	t.Helper()
	r := uv.NewTerminalReader(strings.NewReader(in), "xterm-256color")
	ch := make(chan uv.Event)
	done := make(chan error, 1)
	var p any
	go func() {
		defer close(ch)
		defer func() {
			if p = recover(); p != nil {
				done <- nil
			}
		}()
		done <- r.StreamEvents(context.Background(), ch)
	}()
	for ev := range ch {
		evs = append(evs, fromUltraviolet(ev)...)
	}
	if err := <-done; err != nil {
		t.Fatalf("%q: StreamEvents: %v", in, err)
	}
	return evs, p
}

func fromUltraviolet(ev uv.Event) []event {
	key := func(k uv.Key, release bool) event {
		return event{key: termkey.KeyEvent{
			Code: k.Code, Mod: termkey.Mod(k.Mod), Text: k.Text, ShiftedCode: k.ShiftedCode, //nolint:gosec // ultraviolet's bits fit
			BaseCode: k.BaseCode, IsRepeat: k.IsRepeat, Release: release,
		}}
	}
	switch e := ev.(type) {
	case uv.KeyPressEvent:
		return []event{key(uv.Key(e), false)}
	case uv.KeyReleaseEvent:
		return []event{key(uv.Key(e), true)}
	case uv.PasteEvent:
		return []event{{paste: true, text: e.Content}}
	case uv.MultiEvent:
		var evs []event
		for _, sub := range e {
			evs = append(evs, fromUltraviolet(sub)...)
		}
		return evs
	}
	return nil
}

// TestDecoderMatchesUltraviolet holds MADR D7: every row of termkey's
// corpus gives the same keys and pastes from termkey's Decoder as from
// Bubble Tea's reader, unless differ names the row and why.
func TestDecoderMatchesUltraviolet(t *testing.T) {
	rows := readCorpus(t)
	names := map[string]bool{}
	for _, r := range rows {
		names[r.name] = true
		ours := ourEvents(r.in)
		theirs, panicked := theirEvents(t, r.in)
		reason, listed := differ[r.name]
		same := panicked == nil && reflect.DeepEqual(ours, theirs)
		switch {
		case !same && !listed:
			t.Errorf("%s: %q: termkey gives %+v, ultraviolet %+v (panic: %v)", r.name, r.in, ours, theirs, panicked)
		case same && listed:
			t.Errorf("%s: the decoders agree, but differ lists it (%s)", r.name, reason)
		}
	}
	for name := range differ {
		if !names[name] {
			t.Errorf("differ lists %q, which the corpus has no row of", name)
		}
	}
}
