package cobracmd

import (
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/maccavelli/go-tui-lib/command"
)

// Completion comes from the registry: the values of an enum, for a flag or
// the next positional argument, and the names of commands. Cobra writes
// the scripts for bash, zsh, fish and PowerShell itself, to the writer Run
// gives it (docs/decisions/0006-MADR-command-registry.md A1, A10).

// completionCommand is New's completion command, with one subcommand for
// each shell, writing Cobra's script for it. Cobra's own completion
// command keeps the writer of the first run that creates it
// (completions.go, "out := c.OutOrStdout()"), so a later run of the same
// tree would write its script to the first run's writer; this one writes
// to the run's.
func (b *builder) completionCommand() *cobra.Command {
	verb := map[string]string{annotationNode: nodeVerb}
	cmd := &cobra.Command{
		Use:         completionName,
		Short:       "Generate the completion script for a shell",
		Long:        "Generates the completion script for bash, zsh, fish or PowerShell. Load it in the shell's start-up file.",
		Args:        cobra.ArbitraryArgs,
		Annotations: verb,
		RunE:        b.namespaceRun,
	}
	shells := []struct {
		name string
		gen  func(root *cobra.Command, w io.Writer, desc bool) error
	}{
		{"bash", func(root *cobra.Command, w io.Writer, desc bool) error { return root.GenBashCompletionV2(w, desc) }},
		{"zsh", func(root *cobra.Command, w io.Writer, desc bool) error {
			if desc {
				return root.GenZshCompletion(w)
			}
			return root.GenZshCompletionNoDesc(w)
		}},
		{"fish", func(root *cobra.Command, w io.Writer, desc bool) error { return root.GenFishCompletion(w, desc) }},
		{"powershell", func(root *cobra.Command, w io.Writer, desc bool) error {
			if desc {
				return root.GenPowerShellCompletionWithDesc(w)
			}
			return root.GenPowerShellCompletion(w)
		}},
	}
	for _, sh := range shells {
		var noDesc bool
		sub := &cobra.Command{
			Use:               sh.name,
			Short:             "Generate the completion script for " + sh.name,
			Args:              cobra.NoArgs,
			ValidArgsFunction: cobra.NoFileCompletions,
			Annotations:       verb,
			RunE: func(c *cobra.Command, _ []string) error {
				return sh.gen(c.Root(), c.OutOrStdout(), !noDesc)
			},
		}
		addSwitch(sub.Flags(), &noDesc, "no-descriptions", "leave out the commands' descriptions")
		cmd.AddCommand(sub)
	}
	return cmd
}

// completeArgs completes a registry command's next positional argument:
// an enum's values, or the shell's own completion, such as files, for
// anything else.
func completeArgs(pos []*property, st *state) cobra.CompletionFunc {
	return func(_ *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
		obj, err := placeArgs(pos, st.obj, args)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		for _, p := range pos {
			if p.is(tArray) || obj[p.name] == nil {
				if len(p.enum) > 0 {
					return choices(p), cobra.ShellCompDirectiveNoFileComp
				}
				return nil, cobra.ShellCompDirectiveDefault
			}
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeIDs completes describe's and schema's command: the dotted IDs of
// the commands the shell offers, hidden ones left out.
func (b *builder) completeIDs(_ *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var out []cobra.Completion
	for c := range b.r.Available(b.cfg.context, command.SurfaceCLI) {
		if strings.HasPrefix(string(c.ID), toComplete) {
			out = append(out, cobra.CompletionWithDesc(string(c.ID), c.Title))
		}
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// completeSubcommands completes the help command's words: the commands
// below the ones already given.
func completeSubcommands(cc *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	at, rest, err := cc.Root().Find(args)
	if err != nil || len(rest) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var out []cobra.Completion
	for _, s := range at.Commands() {
		if s.IsAvailableCommand() && s.Name() != helpName && strings.HasPrefix(s.Name(), toComplete) {
			out = append(out, cobra.CompletionWithDesc(s.Name(), s.Short))
		}
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}
