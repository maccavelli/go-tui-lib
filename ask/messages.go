package ask

import "github.com/maccavelli/go-tui-lib/v2/msgcat"

// The Asker's own words, by message ID
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D9).
const (
	// MsgReasonPrompt comes before a reason as it is typed: the answer's
	// label.
	MsgReasonPrompt msgcat.MessageID = "ask.reason-prompt"
	// MsgReasonEcho is an answer with its reason, left in the scrollback:
	// the answer's echo, then the reason.
	MsgReasonEcho msgcat.MessageID = "ask.reason-echo"
	// MsgCanceled is the echo of a question canceled, when its cancel
	// answer has no echo and no label.
	MsgCanceled msgcat.MessageID = "ask.canceled"
)

// Messages is the Asker's words in English, its defaults, by ID: where a
// program starts its translation.
func Messages() msgcat.Table {
	return msgcat.Table{
		MsgReasonPrompt: "%s: ",
		MsgReasonEcho:   "%[1]s: %[2]s",
		MsgCanceled:     "cancel",
	}
}
