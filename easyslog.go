package easyslog

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"sync"
	"time"
)

type (
	// EasySlog is a slog handler that reduces the boilerplate of implementing the
	// slog.Handler boilerplate.
	EasySlog struct {
		formatter Formatter
		leveler   slog.Leveler
		mu        *sync.Mutex
		attrs     []Attr
		groups    []string
		writer    io.Writer
	}

	// Record is passed to the formatter associated with an EasySlog handler. It
	// includes the time, level, PC pointer, and message that the slog.Record
	// holds.
	//
	// Like slog, the `Time` and `PC` fields should be checked to ensure
	// they aren't zero values before use.
	Record struct {
		// The time.Time value provided by slog.Record.
		Time time.Time
		// The level of the current log line.
		Level slog.Level
		// The program counter provided by slog.Record. See slog.Record for more
		// details.
		PC uintptr
		// The message being logged.
		Message string
		// The attributes being logged.
		Attrs []Attr
	}

	// Attr is a flattened slog attribute where `Keys` is a slice of strings
	// representing the groups and key of the attribute logged.
	//
	// e.g. `slog.WithGroup("outer").Info("hello", slog.Group("inner", "a", b"))` would result in `[]string{"outer", "inner", "a"}`
	Attr struct {
		Keys []string
		// The underlying slog.Value attribute being logged. `Resolve` is
		// already called on values that implement `slog.LogValuer``.
		Value slog.Value
	}

	// Formatter is provided the io.Writer of the handler and the Record for the
	// current log line. Each call to `Format` is provided its own buffer which
	// can be written to immediately. If an error is returned the value is not
	// written to the handlers io.Writer.
	Formatter interface {
		Format(w io.Writer, r Record) error
	}

	// Options to configure EasySlog
	Options struct {
		Level slog.Leveler
	}
)

var _ slog.Handler = (*EasySlog)(nil)

// New returns a new EasySlog that delegates the formatting of log lines to the
// provided Formatter.
func New(w io.Writer, formatter Formatter, opts *Options) *EasySlog {
	if opts == nil {
		opts = &Options{}
	}

	return &EasySlog{
		writer:    w,
		formatter: formatter,
		leveler:   slog.LevelDebug,
		mu:        &sync.Mutex{},
	}
}

// Enabled returns if EasySlog handles logs at the given level.
func (handler *EasySlog) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= handler.leveler.Level()
}

// WithAttrs returns a new EasySlog whose attributes are always logged.
func (handler *EasySlog) WithAttrs(slogAttrs []slog.Attr) slog.Handler {
	attrs := make([]Attr, 0, len(slogAttrs))
	for _, attr := range slogAttrs {
		if attr.Value.Any() == nil {
			continue
		}
		attrs = append(attrs, parseValue(attr, handler.groups)...)
	}

	return &EasySlog{
		writer:    handler.writer,
		formatter: handler.formatter,
		leveler:   handler.leveler,
		mu:        handler.mu,
		attrs:     append(handler.attrs, attrs...),
		groups:    handler.groups,
	}
}

// WithGroup returns a new EasySlog that nests all attributes in the provided
// group.
func (handler *EasySlog) WithGroup(name string) slog.Handler {
	if name == "" {
		return handler
	}

	groups := append(handler.groups, name)

	return &EasySlog{
		writer:    handler.writer,
		formatter: handler.formatter,
		leveler:   handler.leveler,
		mu:        handler.mu,
		attrs:     handler.attrs,
		groups:    groups,
	}
}

// Handle converts the slog.Record data into an EasySlog.Record, provides it to
// the formatter, and writes the output to the handlers io.Writer.
func (handler *EasySlog) Handle(_ context.Context, r slog.Record) error {
	values := make([]Attr, 0, r.NumAttrs())
	values = append(values, handler.attrs...)

	r.Attrs(func(a slog.Attr) bool {
		if a.Value.Any() == nil {
			return true
		}
		values = append(values, parseValue(a, handler.groups)...)
		return true
	})

	var buf bytes.Buffer
	err := handler.formatter.Format(&buf, Record{
		Time:    r.Time,
		PC:      r.PC,
		Message: r.Message,
		Level:   r.Level,
		Attrs:   values,
	})

	if err != nil {
		return err
	}

	buf.WriteByte('\n')

	// Lock to protect the writer
	handler.mu.Lock()
	defer handler.mu.Unlock()

	_, err = io.Copy(handler.writer, &buf)
	return err
}

func parseValue(a slog.Attr, keys []string) []Attr {
	if a.Key != "" {
		keys = append(keys, a.Key)
	}

	if a.Value.Kind() != slog.KindGroup {
		return []Attr{{
			Keys:  keys,
			Value: a.Value.Resolve(),
		}}
	}

	values := make([]Attr, 0, len(a.Value.Group()))
	for _, attr := range a.Value.Group() {
		keys := keys
		values = append(values, parseValue(attr, keys)...)
	}

	return values
}
