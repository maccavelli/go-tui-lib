package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"
	"unicode/utf8"
)

// LoadDir's limits (docs/decisions/0014-PLAN-hardening.md Step 5, finding
// H4). Each is an error wrapped as "command: <path>: …"; the files within
// the limits still load.
var (
	// ErrFileTooLarge: a command file is over the size limit. It is not
	// loaded.
	ErrFileTooLarge = errors.New("the file is over the size limit")
	// ErrTooManyFiles: the directory holds more command files than the
	// limit. The ones read before it load, and the walk stops.
	ErrTooManyFiles = errors.New("there are more command files than the limit")
	// ErrTooDeep: a directory is nested deeper than the limit. It is not
	// read.
	ErrTooDeep = errors.New("the directory is nested deeper than the limit")
)

// The default limits LoadDir applies.
const (
	DefaultMaxFileBytes = 256 << 10 // 256 KiB
	DefaultMaxFiles     = 1000      // .md files
	DefaultMaxDepth     = 8         // directory levels below the root
)

// LoadOption sets a limit of LoadDirWith.
type LoadOption interface{ applyLoad(*loadConfig) }

type loadConfig struct {
	maxFileBytes   int64
	maxFiles, maxD int
}

type loadOptionFunc func(*loadConfig)

func (f loadOptionFunc) applyLoad(c *loadConfig) { f(c) }

// WithMaxFileBytes sets the largest command file read, in bytes (default
// DefaultMaxFileBytes). The file is read through a limit, so a size the
// file system reports is never trusted. A value below 1 keeps the default.
func WithMaxFileBytes(n int64) LoadOption {
	return loadOptionFunc(func(c *loadConfig) {
		if n > 0 {
			c.maxFileBytes = n
		}
	})
}

// WithMaxFiles sets how many command files are read (default
// DefaultMaxFiles). A value below 1 keeps the default.
func WithMaxFiles(n int) LoadOption {
	return loadOptionFunc(func(c *loadConfig) {
		if n > 0 {
			c.maxFiles = n
		}
	})
}

// WithMaxDepth sets how many directory levels below the root are read
// (default DefaultMaxDepth): files at the root are at level 0. A value
// below 1 keeps the default.
func WithMaxDepth(n int) LoadOption {
	return loadOptionFunc(func(c *loadConfig) {
		if n > 0 {
			c.maxD = n
		}
	})
}

// LoadDir reads Markdown command files, *.md, from fsys into Prompt
// commands from src, a User, Project or Plugin source. A host passes
// os.Root.FS() for each directory, so a symlink cannot reach outside it;
// where the directories are is the host's choice. Load the result with
// ReplaceSource(src, …).
//
// A file may open with front matter, a strict subset of YAML: one
// key: value per line, from title, description, slash, aliases (comma
// separated), category, danger, when, hidden and arg.<NAME>. An unknown or
// repeated key, a nested value, and a bad danger, slash name or when are
// errors naming the file and line.
//
// The body is the prompt. Each $NAME placeholder ([A-Z][A-Z0-9_]*) is a
// required, positional string argument named as written, in order of first
// use; $ARGUMENTS is the whole slash tail. Expansion substitutes text and
// runs nothing.
//
// The ID is the source's namespace, then the path's parts as ID segments:
// git/commit.md from the user's directory is user.git.commit, with the
// slash name git:commit. A file without front matter is described by its
// first line, and a file that declares no danger is Mutating
// (docs/decisions/0006-MADR-command-registry.md §6, A6). Files and
// directories whose names start with "." are skipped. Each file that cannot
// be read or parsed is an error, and the rest still load.
//
// LoadDir applies the default limits: DefaultMaxFileBytes per file,
// DefaultMaxFiles files and DefaultMaxDepth directory levels. LoadDirWith
// sets others.
func LoadDir(fsys fs.FS, src Source) ([]Command, []error) {
	return LoadDirWith(fsys, src)
}

// LoadDirWith is LoadDir with limits of its own
// (docs/decisions/0014-PLAN-hardening.md Step 5): WithMaxFileBytes,
// WithMaxFiles and WithMaxDepth. A file over the size limit is an error
// wrapping ErrFileTooLarge; a directory past the depth limit is not read,
// with one error wrapping ErrTooDeep; and past the file count the walk
// stops, with one error wrapping ErrTooManyFiles.
func LoadDirWith(fsys fs.FS, src Source, opts ...LoadOption) ([]Command, []error) {
	cfg := loadConfig{maxFileBytes: DefaultMaxFileBytes, maxFiles: DefaultMaxFiles, maxD: DefaultMaxDepth}
	for _, o := range opts {
		o.applyLoad(&cfg)
	}
	ns, err := namespace(src)
	if err == nil && src.Kind != User && src.Kind != Project && src.Kind != Plugin {
		err = fmt.Errorf("command: LoadDir reads user, project and plugin commands, not %s", src.Kind)
	}
	if err != nil {
		return nil, []error{err}
	}
	var (
		cmds  []Command
		errs  []error
		files int
	)
	walk := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			errs = append(errs, fmt.Errorf("command: %s: %w", p, err))
			return nil
		case p != "." && strings.HasPrefix(d.Name(), "."):
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		case d.IsDir() && p != "." && strings.Count(p, "/") >= cfg.maxD:
			errs = append(errs, fmt.Errorf("command: %s: %w (%d levels)", p, ErrTooDeep, cfg.maxD))
			return fs.SkipDir
		case d.IsDir() || path.Ext(p) != ".md":
			return nil
		case files == cfg.maxFiles:
			errs = append(errs, fmt.Errorf("command: %s: %w (%d files)", p, ErrTooManyFiles, cfg.maxFiles))
			return fs.SkipAll
		}
		files++
		c, err := loadFile(fsys, p, src, ns, cfg.maxFileBytes)
		if err != nil {
			errs = append(errs, err)
			return nil
		}
		cmds = append(cmds, c)
		return nil
	})
	if walk != nil {
		errs = append(errs, fmt.Errorf("command: %w", walk))
	}
	return cmds, errs
}

// loadFile reads one command file, of at most limit bytes.
func loadFile(fsys fs.FS, p string, src Source, ns string, limit int64) (Command, error) {
	data, err := readLimited(fsys, p, limit)
	if err != nil {
		return Command{}, fmt.Errorf("command: %s: %w", p, err)
	}
	if !utf8.Valid(data) {
		return Command{}, fmt.Errorf("command: %s: not UTF-8", p)
	}
	fm, body, err := parseFrontMatter(string(data))
	if err != nil {
		if le, ok := errors.AsType[*lineError](err); ok {
			return Command{}, fmt.Errorf("command: %s:%d: %s", p, le.line, le.msg)
		}
		return Command{}, fmt.Errorf("command: %s: %w", p, err)
	}
	stem := strings.TrimSuffix(p, ".md")
	var segs []string
	for part := range strings.SplitSeq(stem, "/") {
		seg := segment(part)
		if seg == "" {
			return Command{}, fmt.Errorf("command: %s: %q has no letter or digit to name the command by", p, part)
		}
		segs = append(segs, seg)
	}
	body = strings.TrimSpace(body)
	c := Command{
		ID:     ID(ns + "." + strings.Join(segs, ".")),
		Title:  stem,
		Slash:  strings.Join(segs, ":"),
		Kind:   Prompt,
		Danger: Mutating,
		Source: src,
	}
	if err := applyFront(&c, fm, body); err != nil {
		return Command{}, fmt.Errorf("command: %s: %w", p, err)
	}
	c.Handler = HandlerFunc(func(_ context.Context, inv *Invocation) (Result, error) {
		values, err := decodeArgs[map[string]string](inv.Args)
		if err != nil {
			return Result{}, err
		}
		text, err := expand(body, values, inv.Raw)
		if err != nil {
			return Result{}, err
		}
		return Result{Text: text}, nil
	})
	return c, nil
}

// readLimited reads file p of fsys through a limit of limit bytes, so the
// size the file system reports is never trusted: one byte more than limit
// is ErrFileTooLarge.
func readLimited(fsys fs.FS, p string, limit int64) ([]byte, error) {
	f, err := fsys.Open(p)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w (%d bytes)", ErrFileTooLarge, limit)
	}
	return data, nil
}

// applyFront sets c's fields from fm, and its arguments and description
// from body.
func applyFront(c *Command, fm frontMatter, body string) error {
	get := func(k string) string { v, _, _ := fm.get(k); return v }
	if v := get("title"); v != "" {
		c.Title = v
	}
	if v := get(keySlash); v != "" {
		c.Slash = v
	}
	c.Description = get("description")
	if c.Description == "" {
		c.Description = firstLine(body)
	}
	c.Category = get("category")
	c.Aliases = splitList(get("aliases"))
	c.When = get("when")
	c.Hidden = get(keyHidden) == "true"
	if v := get("danger"); v != "" {
		d, err := parseDanger(v)
		if err != nil {
			return err
		}
		c.Danger = d
	}
	names := placeholders(body)
	args := make([]stringArg, len(names))
	for i, n := range names {
		args[i] = stringArg{name: n, desc: get("arg." + n), required: true}
	}
	schema, err := stringArgsSchema(args, usesArguments(body))
	if err != nil {
		return err
	}
	c.Args, c.ArgHint = schema, argHint(args)
	return nil
}

// firstLine is body's first non-blank line, without a Markdown heading's
// leading #s.
func firstLine(body string) string {
	for line := range strings.SplitSeq(body, "\n") {
		if line = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "#")); line != "" {
			return line
		}
	}
	return ""
}
