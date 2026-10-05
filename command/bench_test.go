package command

import (
	"fmt"
	"testing"

	"github.com/maccavelli/go-tui-lib/when"
)

// benchRegistry holds n commands, every other one with a When.
func benchRegistry(b *testing.B, n int) *Registry {
	cmds := make([]Command, n)
	for i := range cmds {
		cmds[i] = cmd(ID(fmt.Sprintf("bench.c%04d", i)), UI, func(c *Command) {
			if i%2 == 1 {
				c.When = "workspace.focusedPane == transcript && !workspace.modal"
			}
		})
	}
	return registryOf(b, nil, cmds...)
}

func BenchmarkLookup(b *testing.B) {
	r := benchRegistry(b, 500)
	for b.Loop() {
		r.Lookup("bench.c0250")
	}
}

func BenchmarkAvailable(b *testing.B) {
	r := benchRegistry(b, 500)
	c := when.Map{"workspace.focusedPane": when.StringValue("transcript")}
	for b.Loop() {
		for range r.Available(c, SurfacePalette) {
		}
	}
}

func BenchmarkDispatchLoop(b *testing.B) {
	r := benchRegistry(b, 500)
	req := Request{ID: "bench.c0250", Origin: OriginKey}
	for b.Loop() {
		r.Dispatch(b.Context(), req)
	}
}
