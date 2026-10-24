// Package termkey reads a terminal's keys: the key a terminal sent, its
// modifiers, the key codes and their names, and a Decoder from the bytes a
// terminal sends to keys, pastes and the sequences it does not read as keys
// (0020-MADR-charm-free-cores-glyphs-and-safetext.md D7).
//
// It imports the standard library and the module's internal enum tables,
// and nothing else, so a line-mode CLI reads keys with it and no Bubble
// Tea. Its codes and modifier bits have
// ultraviolet's values, which Bubble Tea's tea.Key takes, and its names are
// the names tea.Key's Keystroke prints, so a program that has both converts
// a tea.Key to a KeyEvent field for field.
//
// The Decoder has no goroutine and no clock. A host reads the terminal,
// gives each read to Decode, and, when Pending reports bytes held and the
// input has been quiet for DefaultEscTimeout or its own choice, calls Flush:
//
//	var d termkey.Decoder
//	ins := d.Decode(buf[:n], nil)
//	if d.Pending() {
//		// after the timeout, with no read between:
//		ins = d.Flush(ins)
//	}
//
// A terminal's reply to a query is one InputUnknown, with its bytes in Raw,
// which termcap.ParseReply reads. A host that probes with termcap does not
// call Flush while its Probe awaits a reply, so a reply split across reads
// is joined, not given up in pieces (0020-MADR A2.4).
//
// Stability: stable. Exported names change only through the deprecation
// policy in AGENTS.md, "API conventions".
package termkey
