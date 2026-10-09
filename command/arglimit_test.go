package command

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
)

// argLimit is DefaultMaxArgBytes, written out so the probe builds before
// the limit existed (docs/decisions/0014-PLAN-hardening.md Step 6).
const argLimit = 1 << 20

// focusArgs is review's arguments of exactly n bytes, n at least 12.
func focusArgs(n int) string { return `{"focus":"` + strings.Repeat("a", n-12) + `"}` }

// overLimit fails t unless err is an *ArgError for n bytes over the limit.
func overLimit(t *testing.T, what string, n int, err error) {
	t.Helper()
	want := fmt.Sprintf("arguments are %d bytes, over %d", n, argLimit)
	if _, ok := errors.AsType[*ArgError](err); !ok || !strings.Contains(err.Error(), want) {
		t.Errorf("%s: %v; want an *ArgError with %q", what, err, want)
	}
}

// TestArgLimit: arguments over the limit are refused before they are read,
// from every way in, and the audit does not record them; arguments
// exactly at it pass (H5). Before they are read means quickly, and with
// under 1 MiB allocated for 16 MiB of arguments (D4).
func TestArgLimit(t *testing.T) {
	type login struct {
		Token string `json:"token" schema:"secret"`
		Note  string `json:"note,omitzero"`
	}
	var log auditLog
	r := registryOf(t, []RegistryOption{WithAuditor(&log)},
		testCommand(t, "review", func(context.Context, *Invocation, reviewArgs) (Result, error) { return Result{}, nil }, WithSlash("review")),
		testCommand(t, "login", func(context.Context, *Invocation, login) (Result, error) { return Result{}, nil }),
		cmd("plain", UI, func(c *Command) { c.Slash = "plain" }),
	)
	const huge = 16 << 20

	t.Run("CallMCP", func(t *testing.T) {
		for name, args := range map[string]string{
			"valid JSON":     focusArgs(huge),
			"not valid JSON": focusArgs(huge)[:huge-2],
		} {
			b := []byte(args)
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			start := time.Now()
			res := r.CallMCP(t.Context(), "review", b, "agent")
			d := time.Since(start)
			runtime.ReadMemStats(&after)
			if d > 100*time.Millisecond {
				t.Errorf("%s: 16 MiB took %v to refuse, over 100ms", name, d)
			}
			if n := after.TotalAlloc - before.TotalAlloc; n >= 1<<20 {
				t.Errorf("%s: refusing 16 MiB allocated %d bytes, 1 MiB or more", name, n)
			}
			want := fmt.Sprintf("arguments are %d bytes, over %d", len(args), argLimit)
			if !res.IsError || len(res.Content) != 1 || !strings.Contains(res.Content[0].Text, want) {
				t.Errorf("%s: %+v; want an error with %q", name, res.Content, want)
			}
			rec := log[len(log)-1]
			if _, ok := errors.AsType[*ArgError](rec.Err); rec.ID != "review" || rec.Args != nil || !ok {
				t.Errorf("%s: the audit record has %d bytes of arguments, %v", name, len(rec.Args), rec.Err)
			}
		}
	})

	t.Run("Run", func(t *testing.T) {
		args := `{"note":"` + strings.Repeat("n", 2<<20) + `","token":"s3cret"}`
		_, err := r.Run(t.Context(), Request{ID: "login", Args: []byte(args), Origin: OriginKey})
		overLimit(t, "a secret in 2 MiB", len(args), err)
		if rec := log[len(log)-1]; rec.Args != nil {
			t.Errorf("a secret in 2 MiB: the audit record has %d bytes of arguments", len(rec.Args))
		}
		_, err = r.Run(t.Context(), Request{ID: "plain", Args: []byte(focusArgs(huge)), Origin: OriginKey})
		overLimit(t, "a command without a schema", huge, err)
		_, err = r.Run(t.Context(), Request{ID: "plain", Raw: strings.Repeat("r", huge), Origin: OriginKey})
		overLimit(t, "16 MiB of Raw", huge, err)
		_, err = r.Run(t.Context(), Request{ID: "review", Args: []byte(focusArgs(argLimit + 1)), Origin: OriginKey})
		overLimit(t, "one byte over", argLimit+1, err)
		if _, err := r.Run(t.Context(), Request{ID: "review", Args: []byte(focusArgs(argLimit)), Origin: OriginKey}); err != nil {
			t.Errorf("exactly at the limit: %v", err)
		}
		if rec := log[len(log)-1]; len(rec.Args) != argLimit {
			t.Errorf("exactly at the limit: the audit record has %d bytes of arguments", len(rec.Args))
		}
		if _, err := r.Run(t.Context(), Request{ID: "plain", Raw: strings.Repeat("r", argLimit), Origin: OriginKey}); err != nil {
			t.Errorf("Raw exactly at the limit: %v", err)
		}
	})

	t.Run("slash", func(t *testing.T) {
		_, err := r.ParseSlash("/review " + strings.Repeat("a ", huge/2))
		overLimit(t, "a 16 MiB tail", huge, err)
		_, err = r.ParseSlash("/plain " + strings.Repeat("r", argLimit+1))
		overLimit(t, "a tail one byte over", argLimit+1, err)
		// The tail fits, and the arguments built from it do not.
		_, err = r.ParseSlash("/review " + strings.Repeat("a", argLimit-11))
		overLimit(t, "arguments one byte over", argLimit+1, err)
		for _, line := range []string{"/review " + strings.Repeat("a", argLimit-12), "/plain " + strings.Repeat("r", argLimit)} {
			req, err := r.ParseSlash(line)
			if err != nil {
				t.Errorf("exactly at the limit: %v", err)
				continue
			}
			if _, err := r.Run(t.Context(), req); err != nil {
				t.Errorf("exactly at the limit, the parsed request does not run: %v", err)
			}
		}
	})

	t.Run("ParseArgs", func(t *testing.T) {
		_, err := r.ParseArgs("review", []string{strings.Repeat("a", huge/2), strings.Repeat("b", huge/2)}, OriginCLI)
		overLimit(t, "16 MiB of words", huge, err)
		// The words fit, and Raw, which joins and quotes them, does not.
		_, err = r.ParseArgs("plain", []string{strings.Repeat("a", argLimit/2), strings.Repeat("b", argLimit/2)}, OriginCLI)
		overLimit(t, "Raw one byte over", argLimit+1, err)
		_, err = r.ParseArgs("review", []string{strings.Repeat("a", argLimit-11)}, OriginCLI)
		overLimit(t, "arguments one byte over", argLimit+1, err)
		for id, words := range map[ID][]string{
			"review": {strings.Repeat("a", argLimit-12)},
			"plain":  {strings.Repeat("a", argLimit/2), strings.Repeat("b", argLimit/2-1)},
		} {
			req, err := r.ParseArgs(id, words, OriginCLI)
			if err != nil {
				t.Errorf("%s exactly at the limit: %v", id, err)
				continue
			}
			if _, err := r.Run(t.Context(), req); err != nil {
				t.Errorf("%s exactly at the limit, the parsed request does not run: %v", id, err)
			}
		}
	})
}

// maskArgs has a secret field, a slice of secrets and a nested object with
// a secret, for FuzzPrepareMask.
type maskArgs struct {
	User   string `json:"user"`
	Token  string `json:"token" schema:"secret"`
	Nested struct {
		Key  string `json:"key" schema:"secret"`
		Note string `json:"note,omitzero"`
	} `json:"nested,omitzero"`
	Keys []string `json:"keys,omitzero" schema:"secret"`
}

// FuzzPrepareMask: prepare and mask never panic; prepare's arguments are
// valid JSON; and mask, of the raw arguments or the prepared ones, is nil
// or has "***" at every secret path and no secret of four bytes or more
// that the input holds only at a secret path
// (docs/decisions/0014-PLAN-hardening.md Step 6).
func FuzzPrepareMask(f *testing.F) {
	for _, s := range []string{
		`{"user":"me","token":"s3cret","nested":{"key":"s3cret"},"keys":["s3cret","k2k2"]}`,
		`{"user":"me","token":"s3cret"}`, `{"user":"s3cret","token":"s3cret"}`, `{"token":{"a":"s3cret"}}`,
		`{"user":"me","token":"t","nested":"s3cret"}`, `{"keys":"s3cret","token":1}`, `{"token":"s3cret"`,
		`["s3cret"]`, `"s3cret"`, ``, `{"user":"me","token":"s3cret","bogus":"s3cret"}`, `{"nested":{"note":"keep"}}`,
	} {
		f.Add([]byte(s))
	}
	c := testCommand(f, "login", func(context.Context, *Invocation, maskArgs) (Result, error) { return Result{}, nil })
	ru, err := compileRule(c.Args)
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		if out, err := ru.prepare(raw); err == nil {
			if !jsontext.Value(out).IsValid() {
				t.Fatalf("%q prepares to %q, which is not valid JSON", raw, out)
			}
			checkMasked(t, out, ru.mask(out))
		}
		checkMasked(t, raw, ru.mask(raw))
	})
}

// checkMasked fails t unless masked, maskArgs's mask of raw, is nil or
// hides every secret raw holds.
func checkMasked(t *testing.T, raw, masked []byte) {
	t.Helper()
	if masked == nil || len(strings.TrimSpace(string(masked))) == 0 {
		return
	}
	var in, out any
	if err := json.Unmarshal(raw, &in); err != nil {
		t.Fatalf("%q does not decode, and masks to %q", raw, masked)
	}
	if err := json.Unmarshal(masked, &out); err != nil {
		t.Fatalf("%q masks to %q, which does not decode: %v", raw, masked, err)
	}
	var secrets, plain []string
	if obj, ok := in.(map[string]any); ok {
		m := out.(map[string]any)
		for k, v := range obj {
			plain = append(plain, k)
			switch n, isObj := v.(map[string]any); {
			case k == "token" || k == "keys":
				secrets = stringsIn(v, secrets)
				if m[k] != secretMask {
					t.Fatalf("%q masks to %q: %s is not %q", raw, masked, k, secretMask)
				}
			case k == "nested" && isObj:
				for k2, v2 := range n {
					plain = append(plain, k2)
					if k2 != "key" {
						plain = stringsIn(v2, plain)
						continue
					}
					secrets = stringsIn(v2, secrets)
					if m[k].(map[string]any)[k2] != secretMask {
						t.Fatalf("%q masks to %q: nested.key is not %q", raw, masked, secretMask)
					}
				}
			default:
				plain = stringsIn(v, plain)
			}
		}
	} else {
		plain = stringsIn(in, plain)
	}
	shown := stringsIn(out, nil)
	for _, s := range secrets {
		if len(s) < 4 || containsAny(plain, s) {
			continue
		}
		if containsAny(shown, s) {
			t.Fatalf("%q masks to %q, which holds the secret %q", raw, masked, s)
		}
	}
}

// stringsIn appends every string in v, objects' keys included.
func stringsIn(v any, out []string) []string {
	switch x := v.(type) {
	case string:
		out = append(out, x)
	case map[string]any:
		for k, c := range x {
			out = stringsIn(c, append(out, k))
		}
	case []any:
		for _, c := range x {
			out = stringsIn(c, out)
		}
	}
	return out
}

// containsAny reports whether any of ss contains s.
func containsAny(ss []string, s string) bool {
	for _, x := range ss {
		if strings.Contains(x, s) {
			return true
		}
	}
	return false
}
