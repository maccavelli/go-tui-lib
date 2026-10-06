package docs_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/command/cobracmd"
	"github.com/maccavelli/go-tui-lib/command/cobracmd/docs"
	"github.com/maccavelli/go-tui-lib/tuitest"
)

type resizeArgs struct {
	Split string `json:"split" arg:"" help:"the split whose separator moves" placeholder:"SPLIT"`
	Delta int    `json:"delta" arg:"" help:"cells to give the pane before it; negative takes" schema:"min=-200,max=200"`
	Quiet bool   `json:"quiet,omitzero" short:"q" help:"say less"`
}

func tree(t *testing.T) *cobra.Command {
	t.Helper()
	r := command.NewRegistry()
	c, err := command.New("workspace.resize", "Move a split", func(context.Context, *command.Invocation, resizeArgs) (command.Result, error) {
		return command.Result{}, nil
	}, command.WithDanger(command.UI), command.WithCategory("Workspace"),
		command.WithDescription("Moves a named split's separator by delta cells."))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Register(c); err != nil {
		t.Fatal(err)
	}
	root, err := cobracmd.New(r, cobracmd.WithName("app"))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

var date = time.Date(2026, time.October, 6, 0, 0, 0, 0, time.UTC)

func TestPagesGolden(t *testing.T) {
	root := tree(t)
	resize, _, err := root.Find([]string{"workspace", "resize"})
	if err != nil {
		t.Fatal(err)
	}
	pages := map[string]func(*cobra.Command) (string, error){
		"man": func(c *cobra.Command) (string, error) {
			var b bytes.Buffer
			err := docs.Man(&b, c, date)
			return b.String(), err
		},
		"md": func(c *cobra.Command) (string, error) {
			var b bytes.Buffer
			err := docs.Markdown(&b, c)
			return b.String(), err
		},
	}
	for kind, page := range pages {
		for name, c := range map[string]*cobra.Command{"root": root, "resize": resize} {
			first, err := page(c)
			if err != nil {
				t.Fatalf("%s %s: %v", kind, name, err)
			}
			second, err := page(c)
			if err != nil || second != first {
				t.Errorf("%s %s changed between two runs: %v", kind, name, err)
			}
			if strings.Contains(first, "Auto generated") || strings.Contains(first, time.Now().Format("2-Jan-2006")) {
				t.Errorf("%s %s carries Cobra's generated-by line or today's date", kind, name)
			}
			if c.DisableAutoGenTag {
				t.Errorf("%s %s left the caller's DisableAutoGenTag set", kind, name)
			}
			tuitest.Text(t, kind+"-"+name, first)
		}
	}
}

func TestNilArguments(t *testing.T) {
	if docs.Man(nil, &cobra.Command{Use: "x"}, date) == nil || docs.Markdown(&bytes.Buffer{}, nil) == nil {
		t.Error("a nil writer or command is not an error")
	}
}
