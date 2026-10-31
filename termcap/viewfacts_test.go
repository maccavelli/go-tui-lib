package termcap

import (
	"reflect"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// The view facts (docs/decisions/0024-PLAN-terminal-text-and-convention-opt-outs.md
// Step 2; 0024-MADR D5 and A3): each view reads its fact first once its
// origin is not NotQueried, and no other view, nor the brand, changes.

// branded is a Caps of brand b outside a multiplexer, whose Kitty keyboard
// query was answered.
func branded(b Brand) Caps {
	var c Caps
	c.Brand.Set(b, Environment)
	c.KittyKeyboard.Set(Supported, Queried)
	return c
}

func over[T any](v T) Fact[T] { return Fact[T]{Value: v, Origin: Override} }

// views is every view termcap derives.
type views struct {
	Links  LinkCaps
	Notify NotifyCaps
	Keys   KeyboardCaps
}

func viewsOf(c Caps) views { return views{c.Links(), c.Notifications(), c.Keyboard()} }

var brands = []Brand{BrandKitty, BrandFoot, BrandAppleTerminal, BrandITerm2, BrandUnknown}

func TestViewFactsChangeTheirViewAlone(t *testing.T) {
	cases := []struct {
		name   string
		set    func(*Caps)
		change func(*views)
	}{
		{"LinksSupport unsupported", func(c *Caps) { c.LinksSupport = over(Unsupported) }, func(v *views) { v.Links.OSC8 = over(Unsupported) }},
		{"LinksSupport supported", func(c *Caps) { c.LinksSupport = over(Supported) }, func(v *views) { v.Links.OSC8 = over(Supported) }},
		{"NotifyOSC9Support", func(c *Caps) { c.NotifyOSC9Support = over(Supported) }, func(v *views) { v.Notify.OSC9 = over(Supported) }},
		{"NotifyOSC777Support", func(c *Caps) { c.NotifyOSC777Support = over(Unsupported) }, func(v *views) { v.Notify.OSC777 = over(Unsupported) }},
		{"KeyboardRequest 0", func(c *Caps) { c.KeyboardRequest = over(0) }, func(v *views) { v.Keys.Request = 0 }},
		{"KeyboardRequest flags", func(c *Caps) { c.KeyboardRequest = over(ansi.KittyReportEventTypes | ansi.KittyReportAlternateKeys) },
			func(v *views) { v.Keys.Request = ansi.KittyReportEventTypes | ansi.KittyReportAlternateKeys }},
		{"PointerSupport", func(c *Caps) { c.PointerSupport = over(Supported) }, func(*views) {}},
		{"ProgressSupport", func(c *Caps) { c.ProgressSupport = over(Supported) }, func(*views) {}},
	}
	for _, b := range brands {
		base := branded(b)
		for _, tc := range cases {
			c := base
			tc.set(&c)
			want := viewsOf(base)
			tc.change(&want)
			if got := viewsOf(c); !reflect.DeepEqual(got, want) {
				t.Errorf("%v, %s:\n got %+v\nwant %+v", b, tc.name, got, want)
			}
			if c.Brand != base.Brand {
				t.Errorf("%v, %s: the brand changed to %+v", b, tc.name, c.Brand)
			}
		}
	}
}

// TestViewFactsNotQueriedChangeNothing: a fact whose origin is NotQueried
// is not set, whatever its value, so every view is today's.
func TestViewFactsNotQueriedChangeNothing(t *testing.T) {
	for _, b := range brands {
		c := branded(b)
		c.LinksSupport.Value, c.NotifyOSC9Support.Value, c.NotifyOSC777Support.Value = Supported, Supported, Unsupported
		c.KeyboardRequest.Value, c.PointerSupport.Value, c.ProgressSupport.Value = 5, Supported, Supported
		if got, want := viewsOf(c), viewsOf(branded(b)); !reflect.DeepEqual(got, want) {
			t.Errorf("%v:\n got %+v\nwant %+v", b, got, want)
		}
	}
}

// TestViewFactsThroughWithOverride: a program sets a view fact through
// WithOverride, and the view follows while the brand stays.
func TestViewFactsThroughWithOverride(t *testing.T) {
	env := Env{"TERM_PROGRAM=Apple_Terminal"}
	plain := EnvCaps(env, "darwin")
	c := EnvCaps(env, "darwin", WithOverride(func(c *Caps) { c.LinksSupport.Set(Supported, Override) }))
	if plain.Links().OSC8.Value != Unsupported {
		t.Fatalf("Apple Terminal's links: %+v, want unsupported", plain.Links().OSC8)
	}
	if got := c.Links().OSC8; got != over(Supported) || c.Brand != plain.Brand {
		t.Errorf("links %+v, brand %+v; want supported by override, brand %+v", got, c.Brand, plain.Brand)
	}
}
