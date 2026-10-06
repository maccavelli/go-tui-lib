package command

import (
	"errors"
	"testing"
)

// FuzzParseSlash: ParseSlash never panics, and a line it accepts runs
// without an argument error, so every accepted line decodes against its
// command's schema.
func FuzzParseSlash(f *testing.F) {
	for _, s := range []string{
		"/resize sidebar 4", "/resize split=sidebar delta=4", `/resize "sidebar" "4"`, "/rs a -200",
		`/add a "b c" force=true d`, "/add", `/review the "quoted" text`, "/plan next", `/resize 'a\' 1`,
		"/resize a=b", "/add force=1", "/resize x 1e3", "/resize \"x\\\"y\" 2",
	} {
		f.Add(s)
	}
	r := slashRegistry(f)
	f.Fuzz(func(t *testing.T, line string) {
		req, err := r.ParseSlash(line)
		if err != nil {
			return
		}
		if _, err := r.Run(t.Context(), req); err != nil {
			if _, ok := errors.AsType[*ArgError](err); ok {
				t.Fatalf("%q parsed to %s, which does not decode: %v", line, req.Args, err)
			}
		}
	})
}
