package inputfiltertea

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/maccavelli/go-tui-lib/v2/inputfilter"
)

// reporter is a model that keeps the mouse and focus messages it gets,
// and quits at want of them.
type reporter struct {
	got  *[]tea.Msg
	want int
}

func (m reporter) Init() tea.Cmd { return nil }

func (m reporter) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.MouseMsg, tea.FocusMsg, tea.BlurMsg:
		*m.got = append(*m.got, msg)
		if len(*m.got) == m.want {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m reporter) View() tea.View { return tea.NewView("") }

// TestReportAgreesWithTea: each report, read by ParseReport and made a
// message by report, is the message Bubble Tea's own reader gives for its
// bytes, through a pipe, so that a report the Reassembler joins reaches a
// model as an unsplit one would.
func TestReportAgreesWithTea(t *testing.T) {
	x10 := func(cb byte, col, row int) string {
		return "\x1b[M" + string([]byte{cb + 32, byte(32 + col), byte(32 + row)})
	}
	raws := []string{
		"\x1b[<0;10;5M", "\x1b[<0;10;5m", "\x1b[<1;2;3M", "\x1b[<2;2;3M", "\x1b[<32;4;4M",
		"\x1b[<35;4;4M", "\x1b[<64;5;6M", "\x1b[<65;5;6M", "\x1b[<66;5;6M", "\x1b[<67;5;6M",
		"\x1b[<128;1;1M", "\x1b[<129;1;1M", "\x1b[<130;1;1M", "\x1b[<131;1;1M",
		"\x1b[<4;7;7M", "\x1b[<8;7;7M", "\x1b[<16;7;7M", "\x1b[<28;300;200M",
		x10(0, 96, 1), x10(1, 10, 2), x10(3, 10, 2), x10(32, 10, 2), x10(65, 223, 223), x10(28, 5, 5),
		"\x1b[I", "\x1b[O",
	}
	var want []tea.Msg
	for _, raw := range raws {
		r, ok := inputfilter.ParseReport(raw)
		if !ok {
			t.Fatalf("ParseReport(%q) refused it", raw)
		}
		want = append(want, report(r))
	}
	rd, wr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer rd.Close()
	defer wr.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var got []tea.Msg
	prog := tea.NewProgram(reporter{got: &got, want: len(raws)}, tea.WithContext(ctx),
		tea.WithInput(rd), tea.WithOutput(io.Discard), tea.WithoutSignalHandler(),
		tea.WithEnvironment([]string{"TERM=xterm-256color"}))
	if _, err := wr.WriteString(strings.Join(raws, "")); err != nil {
		t.Fatal(err)
	}
	if _, err := prog.Run(); err != nil {
		t.Fatalf("Run: %v, after %d messages", err, len(got))
	}
	for i := range raws {
		if i >= len(got) || fmt.Sprintf("%#v", got[i]) != fmt.Sprintf("%#v", want[i]) {
			t.Fatalf("%q: tea gave %#v; ours is %#v", raws[i], at(got, i), want[i])
		}
	}
}

func at(msgs []tea.Msg, i int) tea.Msg {
	if i < len(msgs) {
		return msgs[i]
	}
	return nil
}
