package pick

// Entry is one thing to pick. Fields are added, never changed, so build
// one with a keyed literal.
type Entry struct {
	// ID is the caller's, and stable: what the program acts on. Where
	// palette.FromEntries takes the same list, the IDs are distinct, since
	// the palette keeps one hit per ID.
	ID string
	// Text is matched and drawn: "anthropic/claude-sonnet-5-5", a
	// session's title.
	Text string
	// Detail is drawn after the text, at the right, and not matched:
	// "1M  $3.00/$15.00", "2 hours ago".
	Detail string
}
