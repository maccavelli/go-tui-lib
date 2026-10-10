package glyph

import (
	"reflect"
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

func TestEveryGlyphIsOneCell(t *testing.T) {
	for name, set := range map[string]Set{"Unicode": Unicode(), "ASCII": ASCII()} {
		n := 0
		fields(reflect.ValueOf(set), name, func(path, s string) {
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
		// 4 borders × 13 glyphs, plus 16 single glyphs.
		if n != 68 {
			t.Errorf("%s: walked %d glyphs, want 68", name, n)
		}
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
		fields(reflect.ValueOf(tier.Set()), tier.String(), func(path, s string) {
			n++
			if w := ansi.StringWidth(s); w != 1 || utf8.RuneCountInString(s) != 1 {
				t.Errorf("%s = %q is %d cells and %d runes, want 1 and 1", path, s, w, utf8.RuneCountInString(s))
			}
		})
		if n != 68 {
			t.Errorf("%s: walked %d glyphs, want 68", tier, n)
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
