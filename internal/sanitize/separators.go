package sanitize

// MaxSeparators bounds the parameter separators a caller lets x/ansi's
// parser read. Its parser stores a digit after the last separator with no
// bound on its buffer, and drops the last parameter when the buffer is
// exactly full, so a caller gives it a buffer of MaxSeparators+1 parameters
// and no sequence with MaxSeparators separators or more
// (docs/decisions/0023-MADR-line-mode-components.md A10).
const MaxSeparators = 32

// Separators is the number of parameter separators, ; and :, x/ansi's
// parser reads for the sequence at s's start, on each path to its
// parameter state: ESC, any intermediate bytes 0x20 to 0x2f, then [ or P;
// 0x9B; and 0x90. A sequence of no parameters, a string's, and text have
// none.
func Separators(s string) int {
	i := 0
	switch {
	case s == "":
		return 0
	case s[0] == 0x9b || s[0] == 0x90:
		i = 1
	case s[0] == 0x1b:
		i = 1
		for i < len(s) && s[i] >= 0x20 && s[i] <= 0x2f {
			i++
		}
		if i == len(s) || s[i] != '[' && s[i] != 'P' {
			return 0
		}
		i++
	default:
		return 0
	}
	n := 0
	for ; i < len(s) && s[i] >= 0x30 && s[i] <= 0x3f; i++ {
		if s[i] == ';' || s[i] == ':' {
			n++
		}
	}
	return n
}
