package palette

import (
	"github.com/maccavelli/go-tui-lib/v2/internal/sanitize"
	"github.com/maccavelli/go-tui-lib/v2/msgcat"
)

// The picker's own words, by message ID
// (docs/decisions/0024-MADR-opt-in-and-opt-out-coverage.md D9, A4).
const (
	// MsgCount is the help line's count: the selected row's place, then
	// the rows.
	MsgCount msgcat.MessageID = "palette.count"
	// MsgCountSearching is the count while providers still search: the
	// selected row's place, then the rows. The ellipsis glyph follows it.
	MsgCountSearching msgcat.MessageID = "palette.count-searching"
	// MsgSearching is the count while providers search and no row has
	// come. The ellipsis glyph follows it.
	MsgSearching msgcat.MessageID = "palette.searching"
	// MsgNoResults is the count when nothing matches.
	MsgNoResults msgcat.MessageID = "palette.no-results"
	// MsgArgumentCount is the count in the argument stage: the parameter
	// asked for, then the parameters.
	MsgArgumentCount msgcat.MessageID = "palette.argument-count"
	// MsgFailed is a failed provider's row: its name. The failure glyph
	// comes before it.
	MsgFailed msgcat.MessageID = "palette.failed"
	// MsgRefused is why a value was refused: the parameter's name, then
	// one of the reasons below.
	MsgRefused msgcat.MessageID = "palette.refused"
	// MsgPickOne is the reason when a parameter with a list has no row
	// selected.
	MsgPickOne msgcat.MessageID = "palette.pick-one"
	// MsgRequired is the reason when a required parameter is left empty.
	MsgRequired msgcat.MessageID = "palette.required"
	// MsgNotWholeNumber is the reason when an integer parameter is given
	// something else.
	MsgNotWholeNumber msgcat.MessageID = "palette.not-whole-number"
	// MsgNotNumber is the reason when a number parameter is given
	// something else.
	MsgNotNumber msgcat.MessageID = "palette.not-number"
	// MsgAtLeast is the reason when a number is below its minimum: the
	// minimum.
	MsgAtLeast msgcat.MessageID = "palette.at-least"
	// MsgAtMost is the reason when a number is above its maximum: the
	// maximum.
	MsgAtMost msgcat.MessageID = "palette.at-most"
	// MsgFileLimit is the files provider's last row when it stopped at its
	// limit: the limit.
	MsgFileLimit msgcat.MessageID = "palette.file-limit"
	// MsgFileLimitReason is that row's reason, where a key would be.
	MsgFileLimitReason msgcat.MessageID = "palette.file-limit-reason"
)

// Messages is the picker's words in English, its defaults, by ID: where a
// program starts its translation.
func Messages() msgcat.Table {
	return msgcat.Table{
		MsgCount:           "%[1]d of %[2]d",
		MsgCountSearching:  "%[1]d of %[2]d, searching",
		MsgSearching:       "searching",
		MsgNoResults:       "no results",
		MsgArgumentCount:   "argument %[1]d of %[2]d",
		MsgFailed:          "%s failed",
		MsgRefused:         "%[1]s: %[2]s",
		MsgPickOne:         "pick one of its values",
		MsgRequired:        "required",
		MsgNotWholeNumber:  "not a whole number",
		MsgNotNumber:       "not a number",
		MsgAtLeast:         "at least %s",
		MsgAtMost:          "at most %s",
		MsgFileLimit:       "only the first %d files are searched",
		MsgFileLimitReason: "the file limit",
	}
}

// say is c's text for id, or the default, as one line.
func say(c msgcat.Catalog, id msgcat.MessageID, args ...any) string {
	return sanitize.Line(msgcat.Format(c, id, Messages()[id], args...))
}
