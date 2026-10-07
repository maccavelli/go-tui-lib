package kongcmd

import (
	"slices"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/maccavelli/go-tui-lib/command/cli"
)

// Completion asks the program (docs/decisions/0006-MADR-command-registry.md
// A11): the script for each shell runs "prog __complete <words>", the
// words after the program's name up to the one being completed, which may
// be empty, and Run answers from the parser's model, one candidate a line,
// as the value, a tab and its description. The shell filters them by what
// was typed, and offers files when there are none.

// completeCommand is the word that asks Run for completions.
const completeCommand = "__complete"

// complete writes the candidates for the last of words to the parser's
// stdout.
func (a *Adapter) complete(p *kong.Kong, words []string) int {
	cur := ""
	if len(words) > 0 {
		cur, words = words[len(words)-1], words[:len(words)-1]
	}
	var out strings.Builder
	for _, c := range candidates(p.Model.Node, words, cur) {
		if strings.HasPrefix(c[0], cur) {
			out.WriteString(c[0])
			if c[1] != "" {
				out.WriteString("\t" + strings.ReplaceAll(c[1], "\n", " "))
			}
			out.WriteString("\n")
		}
	}
	if write(p.Stdout, out.String()) != nil {
		return cli.ExitFailed
	}
	return cli.ExitOK
}

// candidates are the values and descriptions that may follow words below
// n: a flag's choices after it, flags when cur starts with a dash, or else
// the commands below and the next positional argument's choices. Hidden
// commands and flags are left out.
func candidates(n *kong.Node, words []string, cur string) [][2]string {
	positional := 0
	var pending *kong.Flag // a flag waiting for its value
	for _, w := range words {
		if pending != nil {
			pending = nil
			continue
		}
		if strings.HasPrefix(w, "-") {
			if f := flagOf(n, w); f != nil && !f.IsBool() && !strings.Contains(w, "=") {
				pending = f
			}
			continue
		}
		if c := child(n, w); c != nil {
			n, positional = c, 0
			continue
		}
		if self := selfChild(n); self != nil {
			n = self
		}
		positional++
	}
	var out [][2]string
	switch {
	case pending != nil:
		for _, e := range enumOf(pending.Value) {
			out = append(out, [2]string{e, ""})
		}
	case strings.HasPrefix(cur, "-"):
		for at := n; at != nil; at = at.Parent {
			for _, f := range at.Flags {
				if !f.Hidden {
					out = append(out, [2]string{"--" + f.Name, f.Help})
				}
			}
		}
	default:
		for _, c := range n.Children {
			if !c.Hidden {
				out = append(out, [2]string{c.Name, c.Help})
			}
		}
		target := n
		if self := selfChild(n); self != nil && positional == 0 {
			target = self
		}
		if positional < len(target.Positional) {
			for _, e := range enumOf(target.Positional[positional]) {
				out = append(out, [2]string{e, ""})
			}
		}
	}
	return out
}

// enumOf is a value's choices, from Kong's enum or, where the adapter
// checks it, the tag's own list.
func enumOf(v *kong.Value) []string {
	if v.Enum != "" {
		return v.EnumSlice()
	}
	return nil
}

// child is the command below n that w names, by name or alias.
func child(n *kong.Node, w string) *kong.Node {
	for _, c := range n.Children {
		if c.Name == w || slices.Contains(c.Aliases, w) {
			return c
		}
	}
	return nil
}

// flagOf is the flag w names, on n or above it.
func flagOf(n *kong.Node, w string) *kong.Flag {
	name, _, _ := strings.Cut(strings.TrimLeft(w, "-"), "=")
	long := strings.HasPrefix(w, "--")
	for at := n; at != nil; at = at.Parent {
		for _, f := range at.Flags {
			if (long && f.Name == name) || (!long && f.Short != 0 && string(f.Short) == name) {
				return f
			}
		}
	}
	return nil
}

// script writes the completion script for shell to the parser's stdout.
func (a *Adapter) script(kctx *kong.Context, shell string) int {
	name := a.cfg.name
	fn := "_" + strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return '_'
	}, name) + "_complete"
	s := scripts[shell]
	s = strings.ReplaceAll(s, "@NAME@", name)
	s = strings.ReplaceAll(s, "@FUNC@", fn)
	if write(kctx.Stdout, s) != nil {
		return cli.ExitFailed
	}
	return cli.ExitOK
}

// scripts are the completion scripts, with @NAME@ for the program's name
// and @FUNC@ for a function name made from it.
var scripts = map[string]string{
	"bash": `# bash completion for @NAME@
@FUNC@() {
    local cur="${COMP_WORDS[COMP_CWORD]}" line out
    out=$("${COMP_WORDS[0]}" __complete "${COMP_WORDS[@]:1:COMP_CWORD}" 2>/dev/null) || return
    COMPREPLY=()
    while IFS= read -r line; do
        line="${line%%$'\t'*}"
        [[ -n $line && $line == "$cur"* ]] && COMPREPLY+=("$line")
    done <<<"$out"
}
complete -o default -F @FUNC@ @NAME@
`,
	"zsh": `#compdef @NAME@
# zsh completion for @NAME@
@FUNC@() {
    local -a items
    local line value
    for line in "${(@f)$("${words[1]}" __complete "${(@)words[2,CURRENT]}" 2>/dev/null)}"; do
        [[ -z $line ]] && continue
        value=${line%%$'\t'*}
        value=${value//:/\\:}
        if [[ $line == *$'\t'* ]]; then
            items+=("$value:${line#*$'\t'}")
        else
            items+=("$value")
        fi
    done
    if (( ${#items} )); then
        _describe '@NAME@' items
    else
        _files
    fi
}
compdef @FUNC@ @NAME@
`,
	"fish": `# fish completion for @NAME@
function @FUNC@
    set -l words (commandline -opc) (commandline -ct)
    $words[1] __complete $words[2..-1] 2>/dev/null
end
complete -c @NAME@ -f -a '(@FUNC@)'
`,
	"powershell": `# PowerShell completion for @NAME@
Register-ArgumentCompleter -Native -CommandName '@NAME@' -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)
    $words = @($commandAst.CommandElements | Select-Object -Skip 1 | ForEach-Object { $_.ToString() })
    if ($wordToComplete -eq '') {
        if ($PSVersionTable.PSVersion -lt [version]'7.3.0') { $words += '""' } else { $words += '' }
    }
    & $commandAst.CommandElements[0].ToString() __complete @words 2>$null | ForEach-Object {
        $value, $desc = $_ -split "` + "`" + `t", 2
        if (-not $desc) { $desc = $value }
        [System.Management.Automation.CompletionResult]::new($value, $value, 'ParameterValue', $desc)
    }
}
`,
}
