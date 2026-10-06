// Package docs writes a Cobra command's man page and Markdown page, the
// same on every run: with the date the caller gives, and without Cobra's
// "Auto generated" line, which carries today's date
// (docs/decisions/0006-MADR-command-registry.md A1).
//
// Each function writes the page of the command it is given, which may be
// any command of a tree; a program writes a tree's pages by calling it for
// each command. Only a program that imports this package compiles
// cobra/doc and the modules it needs.
package docs

import (
	"errors"
	"io"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// Man writes root's man page to w, in section 1, dated date.
func Man(w io.Writer, root *cobra.Command, date time.Time) error {
	if w == nil || root == nil {
		return errors.New("docs: Man needs a writer and a command")
	}
	defer quiet(root)()
	return doc.GenMan(root, &doc.GenManHeader{Date: &date, Section: "1"}, w)
}

// Markdown writes root's Markdown page to w. Its links to other commands'
// pages name them as Cobra does, "app_workspace.md".
func Markdown(w io.Writer, root *cobra.Command) error {
	if w == nil || root == nil {
		return errors.New("docs: Markdown needs a writer and a command")
	}
	defer quiet(root)()
	return doc.GenMarkdownCustom(root, w, func(link string) string { return link })
}

// quiet sets c's DisableAutoGenTag for one page, and returns what puts it
// back, so that the caller's tree is left as it was.
func quiet(c *cobra.Command) func() {
	was := c.DisableAutoGenTag
	c.DisableAutoGenTag = true
	return func() { c.DisableAutoGenTag = was }
}
