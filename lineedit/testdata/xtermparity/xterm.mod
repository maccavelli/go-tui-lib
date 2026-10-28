// The module file lineedit's parity generator is built with, by
// go run -modfile (see main.go). It is not the root's go.mod: the root
// module requires no golang.org/x/term (0023-MADR A5 item 1).
module github.com/maccavelli/go-tui-lib/v2

go 1.27.2

require golang.org/x/term v0.47.0

require golang.org/x/sys v0.49.0 // indirect
