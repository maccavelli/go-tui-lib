package termcap

import (
	"reflect"

	"github.com/maccavelli/go-tui-lib/v2/internal/enum"
	"github.com/maccavelli/go-tui-lib/v2/msgcat"
)

// Disposition is how a finding should be read.
type Disposition uint8

const (
	// Recommendation is advice: the terminal works, with less.
	Recommendation Disposition = iota
	// Issue is something wrong that the user can fix.
	Issue
)

var dispositionNames = enum.Names[Disposition]{
	Pkg: pkgName, Type: "Disposition", Unknown: ErrUnknownName,
	Tokens: []string{"recommendation", "issue"},
}

// String is the disposition's name, as the report and JSON print it.
func (d Disposition) String() string { return dispositionNames.String(d) }

// MarshalText is the disposition's name.
func (d Disposition) MarshalText() ([]byte, error) {
	return dispositionNames.Marshal(d)
}

// UnmarshalText reads a name MarshalText wrote.
func (d *Disposition) UnmarshalText(t []byte) error {
	return dispositionNames.Unmarshal(t, d)
}

// Finding is one thing the doctor tells the user: a reason token, what it
// means, and what would change it. Fix is text, never an action.
type Finding struct {
	ID          string      `json:"id"` // a reason token
	Disposition Disposition `json:"disposition"`
	Message     string      `json:"message"`
	Fix         string      `json:"fix"`
}

// finding is a Finding's disposition and its words, by its token.
type finding struct {
	d            Disposition
	message, fix msgcat.MessageID
}

// findingText says what each reason token means, and what would change
// it, by message ID; Messages has the words.
var findingText = map[string]finding{
	ReasonKittyUnsupported:     {Recommendation, MsgKittyUnsupported, MsgKittyUnsupportedFix},
	ReasonKittyUnknown:         {Issue, MsgKittyUnknown, MsgKittyUnknownFix},
	ReasonLeaksReleases:        {Recommendation, MsgLeaksReleases, MsgLeaksReleasesFix},
	ReasonAlacrittyRelease:     {Recommendation, MsgAlacrittyRelease, MsgAlacrittyReleaseFix},
	ReasonTmuxExtendedKeys:     {Issue, MsgTmuxExtendedKeys, MsgTmuxExtendedKeysFix},
	ReasonMSYSNoKitty:          {Recommendation, MsgMSYSNoKitty, MsgMSYSNoKittyFix},
	ReasonWSLDeadKeys:          {Recommendation, MsgWSLDeadKeys, MsgWSLDeadKeysFix},
	ReasonAppleNoOSC8:          {Recommendation, MsgAppleNoOSC8, MsgAppleNoOSC8Fix},
	ReasonWarpNoOSC8:           {Recommendation, MsgWarpNoOSC8, MsgWarpNoOSC8Fix},
	ReasonTmuxLinks:            {Issue, MsgTmuxLinks, MsgTmuxLinksFix},
	ReasonMuxNoLinks:           {Recommendation, MsgMuxNoLinks, MsgMuxNoLinksFix},
	ReasonUnknownTerminal:      {Recommendation, MsgUnknownTerminal, MsgUnknownTerminalFix},
	ReasonZellijNoForwarding:   {Recommendation, MsgZellijNoForwarding, MsgZellijNoForwardingFix},
	ReasonColorFGBGGuess:       {Recommendation, MsgColorFGBGGuess, MsgColorFGBGGuessFix},
	ReasonDesktopAppearance:    {Recommendation, MsgDesktopAppearance, MsgDesktopAppearanceFix},
	ReasonLegacyConsoleGuess:   {Issue, MsgLegacyConsoleGuess, MsgLegacyConsoleGuessFix},
	ReasonConPTYAnswers:        {Recommendation, MsgConPTYAnswers, MsgConPTYAnswersFix},
	ReasonWindowsTerminalGuess: {Recommendation, MsgWindowsTerminalGuess, MsgWindowsTerminalGuessFix},
	ReasonJetBrainsPaints:      {Recommendation, MsgJetBrainsPaints, MsgJetBrainsPaintsFix},
	ReasonEditorTerminal:       {Recommendation, MsgEditorTerminal, MsgEditorTerminalFix},
	ReasonReplyTooLong:         {Issue, MsgReplyTooLong, MsgReplyTooLongFix},
	ReasonUnknownQuery:         {Issue, MsgUnknownQuery, MsgUnknownQueryFix},
}

// Findings is a finding for each reason token in c, its facts and its
// views, in field order, each token once, in English.
func Findings(c Caps) []Finding { return findingsIn(c, nil) }

// findingsIn is Findings in cat's words.
func findingsIn(c Caps, cat msgcat.Catalog) []Finding {
	var out []Finding
	seen := map[string]bool{}
	add := func(token string) {
		if token == "" || seen[token] {
			return
		}
		seen[token] = true
		f, ok := findingText[token]
		if !ok {
			f = finding{Recommendation, MsgNoDescription, MsgNoDescriptionFix}
		}
		out = append(out, Finding{ID: token, Disposition: f.d, Message: say(cat, f.message), Fix: say(cat, f.fix)})
	}
	v := reflect.ValueOf(c)
	for _, f := range v.Fields() {
		if f.Kind() == reflect.Struct && f.NumField() == 3 && f.Field(2).Kind() == reflect.String {
			add(f.Field(2).String())
		}
	}
	add(c.Keyboard().Kitty.Reason)
	add(c.Links().OSC8.Reason)
	n := c.Notifications()
	add(n.OSC777.Reason)
	add(n.OSC9.Reason)
	return out
}
