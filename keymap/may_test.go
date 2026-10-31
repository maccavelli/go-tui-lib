package keymap

import (
	"testing"

	"github.com/maccavelli/go-tui-lib/v2/termkey"
)

// FuzzMay: entry.may never refuses a key its first stroke Matches, for any
// stroke and key (0023-MADR A12).
func FuzzMay(f *testing.F) {
	f.Add('x', uint16(0), false, 'x', uint16(0), "x", rune(0), false)
	f.Add('a', uint16(termkey.ModShift), false, 'A', uint16(termkey.ModCapsLock), "A", rune(0), false)
	f.Add(termkey.KeyEnter, uint16(0), false, termkey.KeyKpEnter, uint16(0), "", rune(0), true)
	f.Add('q', uint16(termkey.ModCtrl), true, 'й', uint16(termkey.ModCtrl), "", 'q', false)
	// Matched by text alone: "?" typed as shift+/.
	f.Add('?', uint16(0), false, '/', uint16(termkey.ModShift), "?", rune(0), false)
	f.Fuzz(func(t *testing.T, sc rune, sm uint16, scan bool, kc rune, km uint16, text string, base rune, repeat bool) {
		s := Stroke{Code: sc, Mod: termkey.Mod(sm), Scan: scan}
		k := termkey.KeyEvent{Code: kc, Mod: termkey.Mod(km), Text: text, BaseCode: base, IsRepeat: repeat}
		e := &entry{}
		e.first, e.firstText, e.scan = keyName(s.Code), string(s.Code), s.Scan || s.Code == keyLeader
		ev := keyEvent{key: k, name: matchName(k)}
		if s.Matches(k) && !e.may(ev) {
			t.Fatalf("may refused %+v for %+v, which Matches takes", k, s)
		}
	})
}
