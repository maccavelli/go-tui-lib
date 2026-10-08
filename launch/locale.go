package launch

import (
	"strings"

	"github.com/maccavelli/go-tui-lib/glyph"
	"github.com/maccavelli/go-tui-lib/termcap"
)

// tier is the glyph tier env's locale allows
// (docs/decisions/0013-MADR-cli-integration-helpers.md §6). The first set,
// non-empty value of LC_ALL, LC_CTYPE and LANG decides, the precedence
// POSIX gives them: one that names UTF-8 ("utf-8" or "utf8", in any case)
// is TierUnicode, and any other is TierASCII. With none set, it is
// TierUnicode on Windows, which writes to a console in UTF-16 whatever the
// code page and sets none of them, and TierASCII elsewhere, where no locale
// is the C locale. It never gives TierLegacy until theme v2 gives the rule.
func tier(env termcap.Env, goos string) glyph.Tier {
	for _, k := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if v := strings.ToLower(env.Getenv(k)); v != "" {
			if strings.Contains(v, "utf-8") || strings.Contains(v, "utf8") {
				return glyph.TierUnicode
			}
			return glyph.TierASCII
		}
	}
	if goos == "windows" {
		return glyph.TierUnicode
	}
	return glyph.TierASCII
}
