package command

import (
	"context"
	"encoding/json/jsontext"
	"log/slog"
	"time"
)

// Auditor receives a Record for every request the registry is given.
type Auditor interface {
	Audit(Record)
}

// AuditorFunc is a function used as an Auditor.
type AuditorFunc func(Record)

// Audit calls f.
func (f AuditorFunc) Audit(r Record) { f(r) }

// Record is one request's audit entry. Verdict is zero when the request
// failed before the policy was asked. Duration is the handler's, and zero
// when the handler did not run.
type Record struct {
	ID       ID
	Origin   Origin
	Caller   string
	Args     jsontext.Value
	Verdict  Verdict
	Started  time.Time
	Duration time.Duration
	Err      error
}

// SlogAuditor writes each Record to l at level Info, or Warn when it holds
// an error. A nil l discards them. It never uses the default logger.
func SlogAuditor(l *slog.Logger) Auditor {
	if l == nil {
		l = slog.New(slog.DiscardHandler)
	}
	return slogAuditor{l}
}

type slogAuditor struct{ l *slog.Logger }

func (a slogAuditor) Audit(r Record) {
	level := slog.LevelInfo
	attrs := []slog.Attr{
		slog.String("id", string(r.ID)),
		slog.String("origin", r.Origin.String()),
		slog.String("caller", r.Caller),
		slog.String("args", string(r.Args)),
		slog.String("verdict", r.Verdict.ACPKind()),
		slog.Time("started", r.Started),
		slog.Duration("duration", r.Duration),
	}
	if r.Err != nil {
		level = slog.LevelWarn
		attrs = append(attrs, slog.String("err", r.Err.Error()))
	}
	a.l.LogAttrs(context.Background(), level, "command", attrs...)
}
