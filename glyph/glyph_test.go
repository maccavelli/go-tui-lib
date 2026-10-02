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
		// 4 borders × 13 glyphs, plus 12 single glyphs.
		if n != 64 {
			t.Errorf("%s: walked %d glyphs, want 64", name, n)
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
