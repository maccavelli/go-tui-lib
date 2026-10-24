package keymaptea

import (
	"fmt"
	"reflect"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/command"
	"github.com/maccavelli/go-tui-lib/v2/keymap"
	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// teaKeys is every Key constant Bubble Tea v2.0.10's key.go declares, in
// its order: 147, KeyReturn and KeyEsc among them as aliases.
var teaKeys = []struct {
	name string
	code rune
}{
	{"KeyExtended", tea.KeyExtended},
	{"KeyUp", tea.KeyUp},
	{"KeyDown", tea.KeyDown},
	{"KeyRight", tea.KeyRight},
	{"KeyLeft", tea.KeyLeft},
	{"KeyBegin", tea.KeyBegin},
	{"KeyFind", tea.KeyFind},
	{"KeyInsert", tea.KeyInsert},
	{"KeyDelete", tea.KeyDelete},
	{"KeySelect", tea.KeySelect},
	{"KeyPgUp", tea.KeyPgUp},
	{"KeyPgDown", tea.KeyPgDown},
	{"KeyHome", tea.KeyHome},
	{"KeyEnd", tea.KeyEnd},
	{"KeyKpEnter", tea.KeyKpEnter},
	{"KeyKpEqual", tea.KeyKpEqual},
	{"KeyKpMultiply", tea.KeyKpMultiply},
	{"KeyKpPlus", tea.KeyKpPlus},
	{"KeyKpComma", tea.KeyKpComma},
	{"KeyKpMinus", tea.KeyKpMinus},
	{"KeyKpDecimal", tea.KeyKpDecimal},
	{"KeyKpDivide", tea.KeyKpDivide},
	{"KeyKp0", tea.KeyKp0},
	{"KeyKp1", tea.KeyKp1},
	{"KeyKp2", tea.KeyKp2},
	{"KeyKp3", tea.KeyKp3},
	{"KeyKp4", tea.KeyKp4},
	{"KeyKp5", tea.KeyKp5},
	{"KeyKp6", tea.KeyKp6},
	{"KeyKp7", tea.KeyKp7},
	{"KeyKp8", tea.KeyKp8},
	{"KeyKp9", tea.KeyKp9},
	{"KeyKpSep", tea.KeyKpSep},
	{"KeyKpUp", tea.KeyKpUp},
	{"KeyKpDown", tea.KeyKpDown},
	{"KeyKpLeft", tea.KeyKpLeft},
	{"KeyKpRight", tea.KeyKpRight},
	{"KeyKpPgUp", tea.KeyKpPgUp},
	{"KeyKpPgDown", tea.KeyKpPgDown},
	{"KeyKpHome", tea.KeyKpHome},
	{"KeyKpEnd", tea.KeyKpEnd},
	{"KeyKpInsert", tea.KeyKpInsert},
	{"KeyKpDelete", tea.KeyKpDelete},
	{"KeyKpBegin", tea.KeyKpBegin},
	{"KeyF1", tea.KeyF1},
	{"KeyF2", tea.KeyF2},
	{"KeyF3", tea.KeyF3},
	{"KeyF4", tea.KeyF4},
	{"KeyF5", tea.KeyF5},
	{"KeyF6", tea.KeyF6},
	{"KeyF7", tea.KeyF7},
	{"KeyF8", tea.KeyF8},
	{"KeyF9", tea.KeyF9},
	{"KeyF10", tea.KeyF10},
	{"KeyF11", tea.KeyF11},
	{"KeyF12", tea.KeyF12},
	{"KeyF13", tea.KeyF13},
	{"KeyF14", tea.KeyF14},
	{"KeyF15", tea.KeyF15},
	{"KeyF16", tea.KeyF16},
	{"KeyF17", tea.KeyF17},
	{"KeyF18", tea.KeyF18},
	{"KeyF19", tea.KeyF19},
	{"KeyF20", tea.KeyF20},
	{"KeyF21", tea.KeyF21},
	{"KeyF22", tea.KeyF22},
	{"KeyF23", tea.KeyF23},
	{"KeyF24", tea.KeyF24},
	{"KeyF25", tea.KeyF25},
	{"KeyF26", tea.KeyF26},
	{"KeyF27", tea.KeyF27},
	{"KeyF28", tea.KeyF28},
	{"KeyF29", tea.KeyF29},
	{"KeyF30", tea.KeyF30},
	{"KeyF31", tea.KeyF31},
	{"KeyF32", tea.KeyF32},
	{"KeyF33", tea.KeyF33},
	{"KeyF34", tea.KeyF34},
	{"KeyF35", tea.KeyF35},
	{"KeyF36", tea.KeyF36},
	{"KeyF37", tea.KeyF37},
	{"KeyF38", tea.KeyF38},
	{"KeyF39", tea.KeyF39},
	{"KeyF40", tea.KeyF40},
	{"KeyF41", tea.KeyF41},
	{"KeyF42", tea.KeyF42},
	{"KeyF43", tea.KeyF43},
	{"KeyF44", tea.KeyF44},
	{"KeyF45", tea.KeyF45},
	{"KeyF46", tea.KeyF46},
	{"KeyF47", tea.KeyF47},
	{"KeyF48", tea.KeyF48},
	{"KeyF49", tea.KeyF49},
	{"KeyF50", tea.KeyF50},
	{"KeyF51", tea.KeyF51},
	{"KeyF52", tea.KeyF52},
	{"KeyF53", tea.KeyF53},
	{"KeyF54", tea.KeyF54},
	{"KeyF55", tea.KeyF55},
	{"KeyF56", tea.KeyF56},
	{"KeyF57", tea.KeyF57},
	{"KeyF58", tea.KeyF58},
	{"KeyF59", tea.KeyF59},
	{"KeyF60", tea.KeyF60},
	{"KeyF61", tea.KeyF61},
	{"KeyF62", tea.KeyF62},
	{"KeyF63", tea.KeyF63},
	{"KeyCapsLock", tea.KeyCapsLock},
	{"KeyScrollLock", tea.KeyScrollLock},
	{"KeyNumLock", tea.KeyNumLock},
	{"KeyPrintScreen", tea.KeyPrintScreen},
	{"KeyPause", tea.KeyPause},
	{"KeyMenu", tea.KeyMenu},
	{"KeyMediaPlay", tea.KeyMediaPlay},
	{"KeyMediaPause", tea.KeyMediaPause},
	{"KeyMediaPlayPause", tea.KeyMediaPlayPause},
	{"KeyMediaReverse", tea.KeyMediaReverse},
	{"KeyMediaStop", tea.KeyMediaStop},
	{"KeyMediaFastForward", tea.KeyMediaFastForward},
	{"KeyMediaRewind", tea.KeyMediaRewind},
	{"KeyMediaNext", tea.KeyMediaNext},
	{"KeyMediaPrev", tea.KeyMediaPrev},
	{"KeyMediaRecord", tea.KeyMediaRecord},
	{"KeyLowerVol", tea.KeyLowerVol},
	{"KeyRaiseVol", tea.KeyRaiseVol},
	{"KeyMute", tea.KeyMute},
	{"KeyLeftShift", tea.KeyLeftShift},
	{"KeyLeftAlt", tea.KeyLeftAlt},
	{"KeyLeftCtrl", tea.KeyLeftCtrl},
	{"KeyLeftSuper", tea.KeyLeftSuper},
	{"KeyLeftHyper", tea.KeyLeftHyper},
	{"KeyLeftMeta", tea.KeyLeftMeta},
	{"KeyRightShift", tea.KeyRightShift},
	{"KeyRightAlt", tea.KeyRightAlt},
	{"KeyRightCtrl", tea.KeyRightCtrl},
	{"KeyRightSuper", tea.KeyRightSuper},
	{"KeyRightHyper", tea.KeyRightHyper},
	{"KeyRightMeta", tea.KeyRightMeta},
	{"KeyIsoLevel3Shift", tea.KeyIsoLevel3Shift},
	{"KeyIsoLevel5Shift", tea.KeyIsoLevel5Shift},
	{"KeyBackspace", tea.KeyBackspace},
	{"KeyTab", tea.KeyTab},
	{"KeyEnter", tea.KeyEnter},
	{"KeyReturn", tea.KeyReturn},
	{"KeyEscape", tea.KeyEscape},
	{"KeyEsc", tea.KeyEsc},
	{"KeySpace", tea.KeySpace},
}

// teaMods is every modifier Bubble Tea v2.0.10's mod.go declares: 9.
var teaMods = []struct {
	name string
	mod  tea.KeyMod
}{
	{"ModShift", tea.ModShift}, {"ModAlt", tea.ModAlt}, {"ModCtrl", tea.ModCtrl}, {"ModMeta", tea.ModMeta},
	{"ModHyper", tea.ModHyper}, {"ModSuper", tea.ModSuper},
	{"ModCapsLock", tea.ModCapsLock}, {"ModNumLock", tea.ModNumLock}, {"ModScrollLock", tea.ModScrollLock},
}

// TestTeaConversion holds MADR D7: every code and modifier Bubble Tea
// exports converts to termkey and back unchanged, termkey names it as
// tea.Key's Keystroke does, and FromTea and ToTea copy every field of
// tea.Key.
func TestTeaConversion(t *testing.T) {
	if len(teaKeys) != 147 || len(teaMods) != 9 {
		t.Fatalf("%d keys and %d modifiers; key.go and mod.go declare 147 and 9", len(teaKeys), len(teaMods))
	}
	for _, c := range teaKeys {
		k := tea.Key{Code: c.code}
		if got := FromTea(k); got.Code != c.code || ToTea(got) != k || got.Keystroke() != k.Keystroke() {
			t.Errorf("%s: FromTea gives %#v, named %q; ToTea gives back %#v; tea names it %q",
				c.name, got, got.Keystroke(), ToTea(got), k.Keystroke())
		}
	}
	for _, m := range teaMods {
		for _, code := range []rune{'a', tea.KeyUp, tea.KeyLeftShift, tea.KeyRightSuper} {
			k := tea.Key{Code: code, Mod: m.mod}
			if got := FromTea(k); int(got.Mod) != int(m.mod) || ToTea(got) != k || got.Keystroke() != k.Keystroke() {
				t.Errorf("%s with code %d: FromTea gives %#v, named %q; ToTea gives back %#v; tea names it %q", m.name, code, got, got.Keystroke(), ToTea(got), k.Keystroke())
			}
		}
	}
	for _, k := range []tea.Key{
		{Code: 'a', Text: "a"},
		{Code: 'a', ShiftedCode: 'A', Mod: tea.ModShift, Text: "A"},
		{Code: 'ф', ShiftedCode: 'Ф', BaseCode: 'a', Mod: tea.ModShift, Text: "Ф"},
		{Code: 'q', BaseCode: 'a', Mod: tea.ModCtrl | tea.ModAlt | tea.ModCapsLock},
		{Code: 'a', Text: "a", IsRepeat: true},
		{Code: termkey.KeyExtended, Text: "e\xcc\x81"},
		{Code: '/', ShiftedCode: '?', Mod: tea.ModShift, Text: "?"},
	} {
		if got := FromTea(k); ToTea(got) != k || got.Keystroke() != k.Keystroke() || got.String() != k.String() {
			t.Errorf("%#v: FromTea gives %#v, named %q and %q; ToTea gives back %#v; tea names it %q and %q",
				k, got, got.Keystroke(), got.String(), ToTea(got), k.Keystroke(), k.String())
		}
	}
	// Every field of tea.Key is one FromTea copies: a new one fails here.
	if n := reflect.TypeFor[tea.Key]().NumField(); n != 6 {
		t.Errorf("tea.Key has %d fields; FromTea and ToTea copy 6", n)
	}
}

// TestUpdate: a press goes to Key, a release to Key as a release, a
// keymap.TimeoutMsg to Expire, and any other message does nothing; a
// pending sequence's command is the tea.Tick of its Wait, which brings
// back its Timeout.
func TestUpdate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		k := mustNew(t, keymap.Settings{Timeout: 250 * time.Millisecond, Features: keymap.EventTypes, Facts: keymap.Releases}, nil, []keymap.Binding{
			bind(t, keymap.ProgramLayer, command.Global, "x", "cut"),
			bind(t, keymap.ProgramLayer, command.Global, "g g", "top"),
			bind(t, keymap.ProgramLayer, command.Global, "space:release", "talk"),
		})
		d := keymap.NewDispatcher(k)
		if o, cmd := Update(d, press(t, "x"), global, nil); o.Command != "cut" || cmd != nil {
			t.Errorf("x: %+v, a command %v; want cut and no command", o, cmd != nil)
		}
		if o, cmd := Update(d, release(t, "space"), global, nil); o.Command != "talk" || cmd != nil {
			t.Errorf("space released: %+v; want talk", o)
		}
		o, cmd := Update(d, press(t, "g"), global, nil)
		if !o.Handled || o.Wait != 250*time.Millisecond || cmd == nil {
			t.Fatalf("g: %+v, a command %v; want it pending, with the keymap's wait", o, cmd != nil)
		}
		start := time.Now()
		msg := cmd()
		if waited := time.Since(start); waited != 250*time.Millisecond || msg != tea.Msg(o.Timeout) {
			t.Errorf("the timer gave %#v after %v; want the Outcome's Timeout after 250ms", msg, waited)
		}
		if o, cmd := Update(d, msg, global, nil); !o.Handled || fmt.Sprint(o.Replay) != "[g]" || cmd != nil {
			t.Errorf("the timeout: %+v; want g replayed", o)
		}
		if o, cmd := Update(d, tea.WindowSizeMsg{Width: 80}, global, nil); o.Handled || cmd != nil {
			t.Errorf("a WindowSizeMsg: %+v", o)
		}
	})
}

// TestRealTimer: a pending sequence's tea.Tick, with no Timeout set,
// fires after one second, on synctest's fake clock.
func TestRealTimer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := keymap.NewDispatcher(mustNew(t, keymap.Settings{}, nil, []keymap.Binding{bind(t, keymap.ProgramLayer, command.Global, "g g", "top")}))
		_, cmd := Update(d, press(t, "g"), global, nil)
		start := time.Now()
		msg := cmd()
		if waited := time.Since(start); waited != time.Second {
			t.Errorf("the timer fired after %v, want one second", waited)
		}
		if out, _ := Update(d, msg, global, nil); fmt.Sprint(out.Replay) != "[g]" {
			t.Errorf("the real timer's message: %+v; want g replayed", out)
		}
	})
}

// TestReplayIsLossless: a chord that ends without a match replays, through
// ToTea, tea.KeyPressMsgs equal to the ones sent, every field kept.
func TestReplayIsLossless(t *testing.T) {
	d := keymap.NewDispatcher(mustNew(t, keymap.Settings{Features: allFeatures}, nil, []keymap.Binding{
		bind(t, keymap.ProgramLayer, command.Global, "[KeyA] g x", "cmd.x"),
	}))
	sent := []tea.KeyPressMsg{
		{Code: 'ф', ShiftedCode: 'Ф', BaseCode: 'a', Text: "ф"},
		{Code: 'g', Text: "g", IsRepeat: true},
	}
	for _, m := range sent {
		if o, _ := Update(d, m, global, nil); !o.Handled {
			t.Fatalf("%+v: not taken as the chord's stroke: %+v", m, o)
		}
	}
	o, _ := Update(d, press(t, "z"), global, nil)
	var got []tea.KeyPressMsg
	for _, r := range o.Replay {
		got = append(got, tea.KeyPressMsg(ToTea(r)))
	}
	if !reflect.DeepEqual(got, sent) {
		t.Errorf("the replay is %#v; want %#v", got, sent)
	}
}
