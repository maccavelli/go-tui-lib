package termkey

import "testing"

func TestKeystroke(t *testing.T) {
	for _, c := range []struct {
		k    KeyEvent
		want string
	}{
		{KeyEvent{Code: 'a', Text: "a"}, "a"},
		{KeyEvent{Code: 'a', Mod: ModCtrl}, "ctrl+a"},
		{KeyEvent{Code: 'a', Mod: ModShift, ShiftedCode: 'A', Text: "A"}, "shift+a"},
		{KeyEvent{Code: KeyUp, Mod: ModShift | ModAlt | ModCtrl}, "ctrl+alt+shift+up"},
		{KeyEvent{Code: 'x', Mod: ModSuper | ModHyper | ModMeta}, "meta+hyper+super+x"},
		{KeyEvent{Code: 'x', Mod: ModCapsLock | ModNumLock}, "x"},
		{KeyEvent{Code: KeyLeftShift, Mod: ModShift}, "leftshift"},
		{KeyEvent{Code: KeyRightCtrl, Mod: ModCtrl | ModShift}, "shift+rightctrl"},
		{KeyEvent{Code: KeySpace, Text: " "}, "space"},
		{KeyEvent{Code: KeySpace, Mod: ModCtrl}, "ctrl+space"},
		{KeyEvent{Code: KeyEnter}, "enter"},
		{KeyEvent{Code: KeyKpEnter}, "enter"},
		{KeyEvent{Code: KeyF63}, "f63"},
		{KeyEvent{Code: 'ф', Mod: ModCtrl, BaseCode: 'a'}, "ctrl+a"},
		{KeyEvent{Code: KeyExtended, Text: "e\xcc\x81"}, "e\xcc\x81"},
	} {
		if got := c.k.Keystroke(); got != c.want {
			t.Errorf("%+v.Keystroke() = %q, want %q", c.k, got, c.want)
		}
	}
}

func TestKeyString(t *testing.T) {
	for _, c := range []struct {
		k    KeyEvent
		want string
	}{
		{KeyEvent{Code: 'a', Mod: ModShift, ShiftedCode: 'A', Text: "A"}, "A"},
		{KeyEvent{Code: '/', Mod: ModShift, Text: "?"}, "?"},
		{KeyEvent{Code: KeySpace, Text: " "}, "space"},
		{KeyEvent{Code: 'a', Mod: ModCtrl}, "ctrl+a"},
	} {
		if got := c.k.String(); got != c.want {
			t.Errorf("%+v.String() = %q, want %q", c.k, got, c.want)
		}
	}
}

func TestMod(t *testing.T) {
	if s := (ModCtrl | ModShift | ModScrollLock).String(); s != "shift|ctrl|scrolllock" {
		t.Errorf("String = %q", s)
	}
	if s := Mod(0).String(); s != "none" {
		t.Errorf("the empty set is %q", s)
	}
	m := ModCtrl | ModAlt
	if !m.Contains(ModCtrl) || !m.Contains(ModCtrl|ModAlt) || m.Contains(ModCtrl|ModShift) {
		t.Errorf("Contains is wrong for %v", m)
	}
	var back Mod
	if err := back.UnmarshalText([]byte("ctrl|alt")); err != nil || back != m {
		t.Errorf("UnmarshalText(ctrl|alt) = %v, %v; want %v", back, err, m)
	}
	if s := InputPaste.String(); s != "paste" {
		t.Errorf("InputPaste is %q", s)
	}
}
