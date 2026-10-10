package when

import "strconv"

// SyntaxError is Parse's error for a source it cannot read: what is wrong,
// and where, as a byte offset into the source. Offset is -1 when the
// source, or its canonical form, is over MaxSource bytes, which has no one
// place. Err is the error beneath, as a regex's from regexp, or nil.
type SyntaxError struct {
	Offset int
	Msg    string
	Err    error
}

// Error is "when: " and Msg, then ": " and Err's text when there is one,
// then " at offset N" unless Offset is -1.
func (e *SyntaxError) Error() string {
	s := "when: " + e.Msg
	if e.Err != nil {
		s += ": " + e.Err.Error()
	}
	if e.Offset >= 0 {
		s += " at offset " + strconv.Itoa(e.Offset)
	}
	return s
}

// Unwrap is Err.
func (e *SyntaxError) Unwrap() error { return e.Err }
