package keymap

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/command"
)

// normCase is one key, built as its source gives it, and what Normalize
// makes of it on a legacy terminal, with no Disambiguate. Line numbers
// are of ultraviolet v0.0.0-20260811164956-006e29f97886's decoder.go and
// key.go; a key from no decoder is a program's or a test's own tea.Key.
type normCase struct {
	name, source string
	goos         string
	key, want    tea.Key
	always       bool // a rule that holds whatever the features: AltGr and super
}

var normCases = []normCase{
	// C0 codes: ultraviolet already reads them as ctrl and a letter.
	{"0x01 as ultraviolet decodes it", "parseControl, :1158-1161: SOH..SUB, ctrl and the letter", "linux",
		tea.Key{Code: 'a', Mod: tea.ModCtrl}, tea.Key{Code: 'a', Mod: tea.ModCtrl}, false},
	{"a raw 0x01", "no decoder", "linux",
		tea.Key{Code: 0x01}, tea.Key{Code: 'a', Mod: tea.ModCtrl}, false},
	{"a raw 0x0a", "no decoder", "linux",
		tea.Key{Code: 0x0a}, tea.Key{Code: 'j', Mod: tea.ModCtrl}, false},
	{"a raw 0x1c", "no decoder; ultraviolet's FS..US, :1162-1165", "linux",
		tea.Key{Code: 0x1c}, tea.Key{Code: '\\', Mod: tea.ModCtrl}, false},
	{"a raw 0x1f", "no decoder", "linux",
		tea.Key{Code: 0x1f}, tea.Key{Code: '_', Mod: tea.ModCtrl}, false},
	{"a raw NUL", "no decoder; ultraviolet's NUL is ctrl+space, :1128-1132", "linux",
		tea.Key{}, tea.Key{Code: tea.KeySpace, Mod: tea.ModCtrl}, false},
	{"a raw 0x01 under caps lock", "no decoder: a lock is no modifier", "linux",
		tea.Key{Code: 0x01, Mod: tea.ModCapsLock}, tea.Key{Code: 'a', Mod: tea.ModCtrl | tea.ModCapsLock}, false},
	{"a raw 0x01 with alt", "no decoder: the rule is for a C0 code with no modifier", "linux",
		tea.Key{Code: 0x01, Mod: tea.ModAlt}, tea.Key{Code: 0x01, Mod: tea.ModAlt}, false},
	{"a Windows console key with no code", "parseWin32InputKeyEvent, :1856-1860: Code 0, the rune in BaseCode", "windows",
		tea.Key{BaseCode: 0xd83d}, tea.Key{BaseCode: 0xd83d}, false},
	// ESC, TAB and CR keep their names: their codes are the C0 bytes.
	{"ESC", "parseControl, :1145-1149: KeyEscape is 0x1b", "linux",
		tea.Key{Code: tea.KeyEscape}, tea.Key{Code: tea.KeyEscape}, false},
	{"TAB", "parseControl, :1135-1139: KeyTab is 0x09", "linux",
		tea.Key{Code: tea.KeyTab}, tea.Key{Code: tea.KeyTab}, false},
	{"CR", "parseControl, :1140-1144: KeyEnter is 0x0d", "linux",
		tea.Key{Code: tea.KeyEnter}, tea.Key{Code: tea.KeyEnter}, false},
	// BS and DEL.
	{"BS as ultraviolet decodes it", "parseControl, :1133-1134: ctrl+h, which ctrl+h also sends", "linux",
		tea.Key{Code: 'h', Mod: tea.ModCtrl}, tea.Key{Code: 'h', Mod: tea.ModCtrl}, false},
	{"a raw BS", "no decoder", "linux",
		tea.Key{Code: 0x08}, tea.Key{Code: tea.KeyBackspace}, false},
	{"a raw BS with alt", "no decoder", "linux",
		tea.Key{Code: 0x08, Mod: tea.ModAlt}, tea.Key{Code: tea.KeyBackspace, Mod: tea.ModAlt}, false},
	{"DEL with alt", "Decode, :262-266: the ESC prefix; parseControl, :1150-1154: KeyBackspace", "linux",
		tea.Key{Code: tea.KeyBackspace, Mod: tea.ModAlt}, tea.Key{Code: tea.KeyBackspace, Mod: tea.ModAlt}, false},
	{"modifyOtherKeys ctrl+BS", "parseXTermModifyOtherKeys, :1178-1179: KeyBackspace", "linux",
		tea.Key{Code: tea.KeyBackspace, Mod: tea.ModCtrl}, tea.Key{Code: tea.KeyBackspace, Mod: tea.ModCtrl}, false},
	// Upper-case letters with ctrl and no shift.
	{"C1 0x81", "Decode, :290-295: a C1 code is ctrl+alt and code-0x40, upper case", "linux",
		tea.Key{Code: 'A', Mod: tea.ModCtrl | tea.ModAlt}, tea.Key{Code: 'a', Mod: tea.ModCtrl | tea.ModAlt | tea.ModShift}, false},
	{"modifyOtherKeys CSI 27;5;65~", "parseXTermModifyOtherKeys, :1190-1196: the code as sent", "linux",
		tea.Key{Code: 'A', Mod: tea.ModCtrl}, tea.Key{Code: 'a', Mod: tea.ModCtrl | tea.ModShift}, false},
	{"modifyOtherKeys CSI 27;6;65~", "parseXTermModifyOtherKeys: shift already given", "linux",
		tea.Key{Code: 'A', Mod: tea.ModCtrl | tea.ModShift}, tea.Key{Code: 'A', Mod: tea.ModCtrl | tea.ModShift}, false},
	// ctrl+5 and ctrl+4.
	{"modifyOtherKeys ctrl+5", "parseXTermModifyOtherKeys, :1190-1196", "linux",
		tea.Key{Code: '5', Mod: tea.ModCtrl}, tea.Key{Code: ']', Mod: tea.ModCtrl}, false},
	{"Windows console ctrl+5", "parseWin32InputKeyEvent, :1936-1937 the digit as BaseCode; :2033 Code", "windows",
		tea.Key{Code: '5', BaseCode: '5', Mod: tea.ModCtrl}, tea.Key{Code: ']', BaseCode: '5', Mod: tea.ModCtrl}, false},
	{"modifyOtherKeys ctrl+4", "parseXTermModifyOtherKeys, :1190-1196", "linux",
		tea.Key{Code: '4', Mod: tea.ModCtrl}, tea.Key{Code: '\\', Mod: tea.ModCtrl}, false},
	{"modifyOtherKeys ctrl+alt+5", "parseXTermModifyOtherKeys", "linux",
		tea.Key{Code: '5', Mod: tea.ModCtrl | tea.ModAlt}, tea.Key{Code: ']', Mod: tea.ModCtrl | tea.ModAlt}, false},
	{"5 with no ctrl", "parseUtf8", "linux",
		tea.Key{Code: '5', Text: "5"}, tea.Key{Code: '5', Text: "5"}, false},
	// AltGr, on Windows only.
	{"Windows AltGr+q typing @", "parseWin32InputKeyEvent, :2025-2028 left ctrl and right alt; :2034-2045 Text", "windows",
		tea.Key{Code: '@', BaseCode: 'q', Text: "@", Mod: tea.ModCtrl | tea.ModAlt},
		tea.Key{Code: '@', BaseCode: 'q', Text: "@"}, true},
	{"Windows AltGr under caps lock", "parseWin32InputKeyEvent; translateControlKeyState: the lock kept", "windows",
		tea.Key{Code: 'é', Text: "É", Mod: tea.ModCtrl | tea.ModAlt | tea.ModCapsLock},
		tea.Key{Code: 'É', Text: "É", Mod: tea.ModCapsLock}, true},
	{"ctrl+alt with text on linux", "not Windows: no AltGr", "linux",
		tea.Key{Code: '@', Text: "@", Mod: tea.ModCtrl | tea.ModAlt}, tea.Key{Code: '@', Text: "@", Mod: tea.ModCtrl | tea.ModAlt}, false},
	{"Windows ctrl+alt+q, left alt", "parseWin32InputKeyEvent, :2034-2045: no Text without AltGr", "windows",
		tea.Key{Code: 'q', BaseCode: 'q', Mod: tea.ModCtrl | tea.ModAlt},
		tea.Key{Code: 'q', BaseCode: 'q', Mod: tea.ModCtrl | tea.ModAlt}, false},
	// super with stray meta and hyper.
	{"super with meta and hyper", "fromKittyMod, :1347-1366: each Kitty bit its own modifier", "darwin",
		tea.Key{Code: 'k', Mod: tea.ModSuper | tea.ModMeta | tea.ModHyper}, tea.Key{Code: 'k', Mod: tea.ModSuper}, true},
	{"meta and hyper with no super", "fromKittyMod", "linux",
		tea.Key{Code: 'k', Mod: tea.ModMeta | tea.ModHyper}, tea.Key{Code: 'k', Mod: tea.ModMeta | tea.ModHyper}, false},
}

// TestNormalize: each key on a legacy terminal; and with Disambiguate,
// every key unchanged but for AltGr and super.
func TestNormalize(t *testing.T) {
	for _, c := range normCases {
		got, changed := Normalize(c.key, 0, c.goos)
		if got != c.want || changed != (c.want != c.key) {
			t.Errorf("%s (%s): Normalize = %+v, %v; want %+v", c.name, c.source, got, changed, c.want)
		}
		want := c.key
		if c.always {
			want = c.want
		}
		if got, _ := Normalize(c.key, Disambiguate, c.goos); got != want {
			t.Errorf("%s (%s), with Disambiguate: Normalize = %+v; want %+v", c.name, c.source, got, want)
		}
	}
}

// TestDispatchNormalizes: the dispatcher reads keys through Normalize with
// the keymap's features, and replays them as they arrived.
func TestDispatchNormalizes(t *testing.T) {
	ctrl5 := tea.KeyPressMsg{Code: '5', Mod: tea.ModCtrl}
	bs := []Binding{
		bind(t, ProgramLayer, command.Global, "ctrl+]", "cmd.bracket"),
		bind(t, ProgramLayer, command.Global, "ctrl+\\ a", "cmd.chord"),
	}
	d, _ := dispatcher(t, Settings{}, bs...)
	if o, _ := d.Update(ctrl5, global, nil); o.Command != "cmd.bracket" {
		t.Errorf("legacy ctrl+5: %+v; want cmd.bracket", o)
	}
	d, _ = dispatcher(t, Settings{Features: Disambiguate}, bs...)
	if o, _ := d.Update(ctrl5, global, nil); o.Handled {
		t.Errorf("ctrl+5 with Disambiguate: %+v; want nothing", o)
	}
	d, _ = dispatcher(t, Settings{}, bs...)
	if o, _ := d.Update(tea.KeyPressMsg{Code: '4', Mod: tea.ModCtrl}, global, nil); !o.Handled || len(d.Pending()) != 1 {
		t.Fatalf("legacy ctrl+4 does not start ctrl+\\ a: %+v", o)
	}
	o, _ := d.Update(press(t, "z"), global, nil)
	if len(o.Replay) != 1 || o.Replay[0].Code != '4' {
		t.Errorf("the replay is %+v; want ctrl+4 as it arrived", o.Replay)
	}
}

// TestAltGr: on Windows, AltGr text matches no binding, even one for its
// text or its keys, and ends a pending sequence; elsewhere ctrl+alt keys
// bind.
func TestAltGr(t *testing.T) {
	altGr := tea.KeyPressMsg{Code: '@', BaseCode: 'q', Text: "@", Mod: tea.ModCtrl | tea.ModAlt}
	bs := []Binding{
		bind(t, ProgramLayer, command.Global, "@", "cmd.at"),
		bind(t, ProgramLayer, command.Global, "ctrl+alt+@", "cmd.ctrlalt"),
		bind(t, ProgramLayer, command.Global, "ctrl+x n", "cmd.chord"),
	}
	d, _ := dispatcher(t, Settings{GOOS: "windows"}, bs...)
	if o, _ := d.Update(altGr, global, nil); o.Handled || o.Command != "" {
		t.Errorf("AltGr on Windows: %+v; want the pane's", o)
	}
	d.Update(press(t, "ctrl+x"), global, nil)
	o, _ := d.Update(altGr, global, nil)
	if o.Handled || replayed(o) != "[ctrl+x]" || len(d.Pending()) != 0 {
		t.Errorf("AltGr while ctrl+x pends: %+v, pending %v; want ctrl+x replayed", o, d.Pending())
	}
	d, _ = dispatcher(t, Settings{GOOS: "linux"}, bs...)
	if o, _ := d.Update(altGr, global, nil); o.Command != "cmd.ctrlalt" {
		t.Errorf("ctrl+alt+@ on linux: %+v; want cmd.ctrlalt", o)
	}
}
