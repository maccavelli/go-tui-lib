package transcript

import "github.com/maccavelli/go-tui-lib/v2/msgcat"

// The Transcript's own words, by message ID
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D9).
const (
	// MsgEarlierOutput is the line a reprint starts with when it left
	// earlier output out, by the bound in bytes or by the row limit.
	MsgEarlierOutput msgcat.MessageID = "transcript.earlier-output"
)

// Messages is the Transcript's words in English, its defaults, by ID: where
// a program starts its translation.
func Messages() msgcat.Table {
	return msgcat.Table{
		MsgEarlierOutput: "(earlier output is not shown again)",
	}
}
