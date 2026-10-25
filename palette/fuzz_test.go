package palette

import (
	"bytes"
	"testing"
)

// FuzzFrecencyRead: ReadFrom never panics, and a store it read writes what
// reads back to the same store.
func FuzzFrecencyRead(f *testing.F) {
	for _, s := range []string{
		`{"version":1,"entries":[]}`,
		`{"version":1,"entries":[{"id":"a","score":1.5,"at":"2026-10-25T00:00:00Z"},{"id":"b","score":3,"at":"2026-10-24T12:00:00+02:00"}]}`,
		`{"version":2}`,
		`{"version":1,"entries":[{"id":"a","score":1e308}]}`,
		`{"version":1,"entries":[{"id":"a","score":0.000001,"at":"0001-01-01T00:00:00Z"}]}`,
		``,
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, in []byte) {
		fr := NewFrecency()
		if _, err := fr.ReadFrom(bytes.NewReader(in)); err != nil {
			return
		}
		var once, twice bytes.Buffer
		if _, err := fr.WriteTo(&once); err != nil {
			t.Fatalf("WriteTo after a read: %v", err)
		}
		back := NewFrecency()
		if _, err := back.ReadFrom(bytes.NewReader(once.Bytes())); err != nil {
			t.Fatalf("what WriteTo wrote does not read: %v\n%s", err, once.String())
		}
		if _, err := back.WriteTo(&twice); err != nil {
			t.Fatal(err)
		}
		if once.String() != twice.String() {
			t.Fatalf("a read store wrote back otherwise:\n%s\n%s", once.String(), twice.String())
		}
	})
}
