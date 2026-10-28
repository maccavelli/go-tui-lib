package pick

import "github.com/maccavelli/go-tui-lib/v2/msgcat"

// The Chooser's own words, by message ID
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D9).
const (
	// MsgCount is the count after the query: the selected row's place, then
	// the matches.
	MsgCount msgcat.MessageID = "pick.count"
	// MsgCountMore is the count when the index yielded as many candidates
	// as it yields at most, so more may match: the selected row's place,
	// then the matches.
	MsgCountMore msgcat.MessageID = "pick.count-more"
	// MsgNoResults is the count when nothing matches.
	MsgNoResults msgcat.MessageID = "pick.no-results"
	// MsgCanceled is left after the prompt when the pick is canceled.
	MsgCanceled msgcat.MessageID = "pick.canceled"
)

// Messages is the Chooser's words in English, its defaults, by ID: where a
// program starts its translation.
func Messages() msgcat.Table {
	return msgcat.Table{
		MsgCount:     "%[1]d of %[2]d",
		MsgCountMore: "%[1]d of %[2]d+",
		MsgNoResults: "no results",
		MsgCanceled:  "cancel",
	}
}
