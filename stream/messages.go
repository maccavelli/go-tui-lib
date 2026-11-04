package stream

import "github.com/maccavelli/go-tui-lib/v2/msgcat"

// stream's own words, by message ID
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D9).
const (
	// MsgOmitted is the line a Bounded shows in place of the lines it left
	// out, given their count.
	MsgOmitted msgcat.MessageID = "stream.omitted"
)

// Messages is stream's words in English, its defaults, by ID: where a
// program starts its translation.
func Messages() msgcat.Table {
	return msgcat.Table{
		MsgOmitted: "(%d lines not shown)",
	}
}
