package glyph

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

// fields walks a Set, or any struct of strings and nested structs, by
// reflection, so a field added later is checked without editing this test.
func fields(v reflect.Value, path string, visit func(path, s string)) {
	switch v.Kind() {
	case reflect.Struct:
		for i := range v.NumField() {
			fields(v.Field(i), path+"."+v.Type().Field(i).Name, visit)
		}
	case reflect.String:
		visit(path, v.String())
	default:
		panic("glyph: unexpected field kind " + v.Kind().String() + " at " + path)
	}
}

// glyphs walks a Set's glyphs, Keys left out: its ASCII twins are words
// (docs/decisions/0007-MADR-keymap-engine.md A8), which TestKeyGlyphs
// checks.
func glyphs(set Set, path string, visit func(path, s string)) {
	fields(reflect.ValueOf(set), path, func(p, s string) {
		if !strings.HasPrefix(p, path+".Keys.") {
			visit(p, s)
		}
	})
}

func TestEveryGlyphIsOneCell(t *testing.T) {
	for name, set := range map[string]Set{"Unicode": Unicode(), "ASCII": ASCII()} {
		n := 0
		glyphs(set, name, func(path, s string) {
			n++
			if s == "" {
				t.Errorf("%s is empty", path)
			}
			if w := ansi.StringWidth(s); w != 1 {
				t.Errorf("%s = %q is %d cells, want 1", path, s, w)
			}
			if utf8.RuneCountInString(s) != 1 {
				t.Errorf("%s = %q is %d runes, want 1", path, s, utf8.RuneCountInString(s))
			}
		})
		// 4 borders × 13 glyphs, plus 26 single glyphs.
		if n != 78 {
			t.Errorf("%s: walked %d glyphs, want 78", name, n)
		}
	}
}

// TestOneCellInBothMethods: every glyph is one cell whichever way the
// terminal measures, before and after it reports grapheme clustering (mode
// 2027), so a workspace that switches method lays a view out the same.
func TestOneCellInBothMethods(t *testing.T) {
	for name, set := range map[string]Set{"Unicode": Unicode(), "ASCII": ASCII()} {
		glyphs(set, name, func(path, s string) {
			for _, m := range []ansi.Method{ansi.WcWidth, ansi.GraphemeWidth} {
				if w := m.StringWidth(s); w != 1 {
					t.Errorf("%s = %q is %d cells under method %d, want 1", path, s, w, m)
				}
			}
		})
	}
}

// TestPromptAndMask: the input prompt and the secret mask a picker draws in
// place of bubbles' "> " and '*' (docs/decisions/0008-MADR-command-palette.md
// A3 item 8) differ between the sets, so the ASCII set is a real twin.
func TestPromptAndMask(t *testing.T) {
	u, a := Unicode(), ASCII()
	if u.Prompt == a.Prompt || u.Mask == a.Mask {
		t.Errorf("Prompt %q/%q, Mask %q/%q: a Unicode glyph equals its ASCII twin", u.Prompt, a.Prompt, u.Mask, a.Mask)
	}
	if a.Prompt != ">" || a.Mask != "*" {
		t.Errorf("ASCII Prompt %q, Mask %q; want > and *", a.Prompt, a.Mask)
	}
}

// TestStatusAndMeter pins the status, heading, meter and control glyphs
// of both sets (docs/decisions/0020-MADR-charm-free-cores-glyphs-and-safetext.md
// D8): each field's two forms differ, and the four status glyphs are four
// shapes in each set, so a state never rests on colour alone.
func TestStatusAndMeter(t *testing.T) {
	type row struct{ done, failed, running, pending, heading, fill, track, control string }
	get := func(s Set) row {
		return row{s.Done, s.Failed, s.Running, s.Pending, s.Heading, s.MeterFill, s.MeterTrack, s.Control}
	}
	want := map[string]row{
		"Unicode": {"✓", "✗", "●", "○", "▸", "━", "╌", "␛"},
		"ASCII":   {"+", "x", "*", "-", ">", "#", "-", "^"},
	}
	sets := map[string]Set{"Unicode": Unicode(), "ASCII": ASCII()}
	for name, set := range sets {
		if got := get(set); got != want[name] {
			t.Errorf("%s: %+v, want %+v", name, got, want[name])
		}
		status := []string{set.Done, set.Failed, set.Running, set.Pending}
		for i, a := range status {
			for _, b := range status[i+1:] {
				if a == b {
					t.Errorf("%s: the status glyph %q stands for two states", name, a)
				}
			}
		}
	}
	u, a := reflect.ValueOf(get(Unicode())), reflect.ValueOf(get(ASCII()))
	for i := range u.NumField() {
		if u.Field(i).String() == a.Field(i).String() {
			t.Errorf("%s is %q in both sets", u.Type().Field(i).Name, u.Field(i).String())
		}
	}
}

// TestControlIsNotEmoji pins Control's Unicode form, U+241B SYMBOL FOR
// ESCAPE. Checked against Unicode 18.0's emoji-data.txt when it was
// chosen, it has no emoji property, so no terminal draws it two cells wide
// as an emoji presentation (0020-PLAN Step 3); a replacement is checked the
// same way before it changes this test.
func TestControlIsNotEmoji(t *testing.T) {
	if got := Unicode().Control; got != "␛" {
		t.Errorf("Unicode().Control = %q (%U), want U+241B, the code point checked against emoji-data.txt", got, []rune(got))
	}
}

func TestASCIISetIsASCII(t *testing.T) {
	fields(reflect.ValueOf(ASCII()), "ASCII", func(path, s string) {
		for _, r := range s {
			if r > 0x7e || r < 0x20 {
				t.Errorf("%s = %q holds %U, which is not printable ASCII", path, s, r)
			}
		}
	})
}

// TestTierWidths holds every tier's table to the same rule as Unicode() and
// ASCII(): each glyph one rune, one cell wide, so a view laid out for one
// tier fits every other.
func TestTierWidths(t *testing.T) {
	for _, tier := range []Tier{TierUnicode, TierLegacy, TierASCII} {
		n := 0
		glyphs(tier.Set(), tier.String(), func(path, s string) {
			n++
			if w := ansi.StringWidth(s); w != 1 || utf8.RuneCountInString(s) != 1 {
				t.Errorf("%s = %q is %d cells and %d runes, want 1 and 1", path, s, w, utf8.RuneCountInString(s))
			}
		})
		if n != 78 {
			t.Errorf("%s: walked %d glyphs, want 78", tier, n)
		}
	}
}

func TestTierText(t *testing.T) {
	for _, c := range []struct {
		tier Tier
		text string
		set  Set
	}{
		{TierUnicode, "unicode", Unicode()},
		// The legacy tier is ASCII until theme v2 fills it.
		{TierLegacy, "legacy", ASCII()},
		{TierASCII, "ascii", ASCII()},
	} {
		if got := c.tier.String(); got != c.text {
			t.Errorf("Tier(%d).String() = %q, want %q", c.tier, got, c.text)
		}
		b, err := c.tier.MarshalText()
		if err != nil || string(b) != c.text {
			t.Errorf("Tier(%d).MarshalText() = %q, %v", c.tier, b, err)
		}
		var back Tier
		if err := back.UnmarshalText([]byte(c.text)); err != nil || back != c.tier {
			t.Errorf("UnmarshalText(%q) = %d, %v", c.text, back, err)
		}
		if c.tier.Set() != c.set {
			t.Errorf("%s.Set() is not the table the tier names", c.tier)
		}
	}
	if TierUnicode != 0 {
		t.Error("TierUnicode is not the zero Tier")
	}
	if got := Tier(9).String(); got != "Tier(9)" {
		t.Errorf("Tier(9).String() = %q", got)
	}
	if _, err := Tier(9).MarshalText(); err == nil || err.Error() != "glyph: Tier 9 has no name" {
		t.Errorf("Tier(9).MarshalText(): %v", err)
	}
	if Tier(9).Set() != ASCII() {
		t.Error("an unnamed tier is not ASCII")
	}
	v := TierASCII
	if err := v.UnmarshalText([]byte("Unicode")); err == nil || err.Error() != `glyph: unknown Tier "Unicode"` || v != TierASCII {
		t.Errorf("UnmarshalText(\"Unicode\"): %v, left %s", err, v)
	}
}

// TestKeyGlyphs: every tier gives each key label; a Unicode form is one
// rune and one cell, an ASCII twin printable ASCII, a modifier's ending in
// "+"; and the two sets differ in each.
func TestKeyGlyphs(t *testing.T) {
	modifiers := map[string]bool{"Option": true, "Command": true, "Control": true, "Shift": true}
	for _, tier := range []Tier{TierUnicode, TierLegacy, TierASCII} {
		n := 0
		fields(reflect.ValueOf(tier.Set().Keys), tier.String()+".Keys", func(path, s string) {
			n++
			name := path[strings.LastIndex(path, ".")+1:]
			switch {
			case s == "":
				t.Errorf("%s is empty", path)
			case tier == TierUnicode && (ansi.StringWidth(s) != 1 || utf8.RuneCountInString(s) != 1):
				t.Errorf("%s = %q is %d cells and %d runes, want 1 and 1", path, s, ansi.StringWidth(s), utf8.RuneCountInString(s))
			case tier != TierUnicode && modifiers[name] != strings.HasSuffix(s, "+"):
				t.Errorf("%s = %q: a modifier's word ends in +, and only a modifier's", path, s)
			}
			for _, r := range s {
				if tier != TierUnicode && (r > 0x7e || r < 0x20) {
					t.Errorf("%s = %q holds %U, which is not printable ASCII", path, s, r)
				}
			}
		})
		if n != 8 {
			t.Errorf("%s: walked %d key glyphs, want 8", tier, n)
		}
	}
	u, a := reflect.ValueOf(Unicode().Keys), reflect.ValueOf(ASCII().Keys)
	for i := range u.NumField() {
		if u.Field(i).String() == a.Field(i).String() {
			t.Errorf("Keys.%s is %q in both sets", u.Type().Field(i).Name, u.Field(i).String())
		}
	}
}

func TestFor(t *testing.T) {
	if For(true) != Unicode() || For(false) != ASCII() {
		t.Fatal("For does not choose the set by utf8")
	}
}

func TestUnicodeBordersDiffer(t *testing.T) {
	u := Unicode()
	if u.Light.TopLeft == u.Rounded.TopLeft || u.Light.Top == u.Heavy.Top || u.Light.Top == u.Double.Top {
		t.Fatal("the Unicode border styles are not distinct")
	}
}
