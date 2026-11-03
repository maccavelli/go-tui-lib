package inputfilter

import (
	"net/url"
	"strings"
)

// ClipboardImageRequestMsg is an empty bracketed paste, which Windows
// Terminal before 1.25 sends for a clipboard that holds only an image
// (docs/reports/0003-REPORT-agent-tui-ecosystem-research.md §8.5): the
// program reads the image itself. Flusher.ClipboardImageRequest tells
// one.
type ClipboardImageRequestMsg struct{}

// NormalizePaste makes a paste's CR LF and lone CR an LF, since ConPTY
// gives line ends as CR alone.
func NormalizePaste(s string) string {
	if !strings.Contains(s, "\r") {
		return s
	}
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}

// PastedPath reads s as a dropped or pasted path, as codex's
// normalize_pasted_path does (codex tui/src/clipboard_paste.rs:259-369):
// trimmed, its matching quotes removed, and a file:// URL on this host
// decoded. It is a path only when it begins with a path's anchor, "/",
// "~/", a drive ("C:\" or "C:/") or a UNC "\\", as grok's
// starts_with_drop_anchor has it, so that prose is refused. Where goos is
// not "windows", a backslash escapes the byte after it, and an unquoted
// space not escaped is two paths, refused; a drive or UNC path is read as
// it is.
func PastedPath(s, goos string) (string, bool) {
	s = strings.TrimSpace(s)
	quoted := false
	if unq, ok := unquote(s); ok {
		s, quoted = unq, true
	}
	if strings.HasPrefix(s, "file://") {
		return fileURL(s, goos)
	}
	if windowsPath(s) {
		return s, true
	}
	if !strings.HasPrefix(s, "/") && !strings.HasPrefix(s, "~/") {
		return "", false
	}
	if goos == goosWindows {
		if !quoted && strings.ContainsRune(s, ' ') {
			return "", false
		}
		return s, true
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '\\' && i+1 < len(s):
			i++
			b.WriteByte(s[i])
		case c == '\\':
			return "", false
		case c == ' ' && !quoted:
			return "", false
		default:
			b.WriteByte(c)
		}
	}
	return b.String(), true
}

// unquote is s without its matching outer quotes, single or double.
func unquote(s string) (string, bool) {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1], true
	}
	return s, false
}

// fileURL is a file:// URL's path, on this host: on Windows, "/C:/x" is
// read as the drive's "C:\x".
func fileURL(s, goos string) (string, bool) {
	u, err := url.Parse(s)
	if err != nil || u.Scheme != "file" || u.Host != "" && u.Host != "localhost" || u.Path == "" {
		return "", false
	}
	p := u.Path
	if goos == goosWindows && len(p) >= 3 && p[0] == '/' && windowsPath(p[1:]) {
		return strings.ReplaceAll(p[1:], "/", `\`), true
	}
	return p, true
}

// windowsPath reports whether s begins as a drive's path, "C:\" or "C:/",
// or a UNC one, "\\".
func windowsPath(s string) bool {
	if strings.HasPrefix(s, `\\`) {
		return true
	}
	return len(s) >= 3 && (s[0]|0x20 >= 'a' && s[0]|0x20 <= 'z') && s[1] == ':' && (s[2] == '\\' || s[2] == '/')
}

// dropAnchor reports whether s begins as a dropped path does: a path's
// anchor or a file:// URL, quoted or not (grok
// xai-grok-pager-render/src/prompt_images.rs:939-960).
func dropAnchor(s string) bool {
	anchored := func(s string) bool {
		return strings.HasPrefix(s, "/") || strings.HasPrefix(s, "~/") || windowsPath(s) || strings.HasPrefix(s, "file://")
	}
	if anchored(s) {
		return true
	}
	unq, ok := unquote(s)
	return ok && anchored(unq)
}
