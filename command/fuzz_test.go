package command

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzParseArgs: ParseArgs never panics; arguments it accepts are already
// prepared, so preparing them again changes nothing; and its Raw splits
// back into the same words, which parse to the same request
// (docs/decisions/0014-PLAN-component-native-forms.md Step 5). The input is
// the words, one per line.
func FuzzParseArgs(f *testing.F) {
	for _, s := range []string{
		"h", "h\n--count=3\n-v", "--count\n3\nh\na\nb", "h\n--\n-v\n--count=2", "-n\n3\nh", "h\n-",
		"--mode\nfast\nh", "--tags=a\nh\n--tags\nb", "h h", "it's\n\"q\"\n$HOME", "", "--bogus",
		"h\n--wait=1m30s", "fix\nthe\ntests", "s\n--\n-3", "--verbose=false\nh", "\n\n", "a\tb\n''",
	} {
		f.Add(s)
	}
	r := cliRegistry(f)
	snap := r.snap.Load()
	f.Fuzz(func(t *testing.T, s string) {
		var args []string
		if s != "" {
			args = strings.Split(s, "\n")
		}
		for _, id := range []ID{"deploy", "review", "files.add", "workspace.resize"} {
			req, err := r.ParseArgs(id, args, OriginCLI)
			if err != nil {
				continue
			}
			again, err := snap.entries[snap.byID[id]].args.prepare(req.Args)
			if err != nil || string(again) != string(req.Args) {
				t.Fatalf("%s %q: Args %s prepare to %s, %v", id, args, req.Args, again, err)
			}
			if !utf8.ValidString(s) {
				continue // splitting reads runes, so invalid UTF-8 does not survive Raw
			}
			words, err := splitWords(req.Raw)
			if err != nil {
				t.Fatalf("%s %q: Raw %q does not split: %v", id, args, req.Raw, err)
			}
			back := make([]string, len(words))
			for i, w := range words {
				back[i] = w.text
			}
			if !slices.Equal(back, args) && (len(back) != 0 || len(args) != 0) {
				t.Fatalf("%s %q: Raw %q splits to %q", id, args, req.Raw, back)
			}
			req2, err := r.ParseArgs(id, back, OriginCLI)
			if err != nil || string(req2.Args) != string(req.Args) || req2.Raw != req.Raw {
				t.Fatalf("%s %q: parsing Raw's words gives %s %q, %v", id, args, req2.Args, req2.Raw, err)
			}
		}
	})
}

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
