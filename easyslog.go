package easyslog

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"sync"
	"time"

	"slices"
)

type (
	// EasySlog is a slog handler that reduces the boilerplate of implementing the
	// slog.Handler boilerplate.
	EasySlog struct {
		formatter    Formatter
		leveler      slog.Leveler
		mu           *sync.Mutex
		attrs        []Attr
		writer       io.Writer
		groupIndices []int
		root         *Attr
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
		Attrs []*Attr
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
		opts = &Options{
			Level: slog.LevelInfo,
		}
	}

	root := &Attr{
		Key:      "",
		Value:    slog.AnyValue(nil),
		Children: make([]*Attr, 0),
	}

	return &EasySlog{
		root:         root,
		writer:       w,
		formatter:    formatter,
		leveler:      opts.Level,
		groupIndices: []int{},
		mu:           &sync.Mutex{},
	}
}

// Enabled returns if EasySlog handles logs at the given level.
func (handler *EasySlog) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= handler.leveler.Level()
}

func (handler *EasySlog) getCurrentGroup(root *Attr) *Attr {
	group := root
	for _, i := range handler.groupIndices {
		group = group.Children[i]
	}

	return group
}

// WithAttrs returns a new EasySlog whose attributes are always logged.
func (handler *EasySlog) WithAttrs(slogAttrs []slog.Attr) slog.Handler {
	root := handler.root.clone()

	for _, attr := range slogAttrs {
		if attr.Value.Any() == nil {
			continue
		}
		parseValue(attr, handler.getCurrentGroup(root))
	}

	return &EasySlog{
		writer:       handler.writer,
		formatter:    handler.formatter,
		leveler:      handler.leveler,
		mu:           handler.mu,
		groupIndices: handler.groupIndices,
		root:         root,
	}
}

// WithGroup returns a new EasySlog that nests all attributes in the provided
// group.
func (handler *EasySlog) WithGroup(name string) slog.Handler {
	if name == "" {
		return handler
	}

	group := &Attr{
		Key:      name,
		Value:    slog.AnyValue(nil),
		Children: make([]*Attr, 0),
	}

	root := handler.root.clone()
	currentGroup := handler.getCurrentGroup(root)
	currentGroup.Children = append(currentGroup.Children, group)

	return &EasySlog{
		writer:       handler.writer,
		formatter:    handler.formatter,
		leveler:      handler.leveler,
		mu:           handler.mu,
		attrs:        handler.attrs,
		groupIndices: append(handler.groupIndices, len(currentGroup.Children)-1),
		root:         root,
	}
}

// Handle converts the slog.Record data into an EasySlog.Record, provides it to
// the formatter, and writes the output to the handlers io.Writer.
func (handler *EasySlog) Handle(_ context.Context, r slog.Record) error {
	root := handler.root.clone()
	currentGroup := handler.getCurrentGroup(root)

	r.Attrs(func(a slog.Attr) bool {
		parseValue(a, currentGroup)
		return true
	})

	prune(root)

	rootAttrs := make([]*Attr, 0, len(root.Children))
	for _, attr := range root.Children {
		rootAttrs = append(rootAttrs, attr)
	}

	var buf bytes.Buffer
	err := handler.formatter.Format(&buf, Record{
		Time:    r.Time,
		PC:      r.PC,
		Message: r.Message,
		Level:   r.Level,
		Attrs:   rootAttrs,
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

func parseValue(a slog.Attr, parent *Attr) {
	if a.Value.Kind() != slog.KindGroup && a.Value.Any() == nil {
		return
	}

	if a.Value.Kind() != slog.KindGroup {
		parent.Children = append(parent.Children, &Attr{
			Key:   a.Key,
			Value: a.Value.Resolve(),
		})

		return
	}

	groupAttr := parent
	isSubgroup := false
	if a.Key != "" {
		isSubgroup = true
		groupAttr = &Attr{
			Key:      a.Key,
			Value:    slog.AnyValue(nil),
			Children: make([]*Attr, 0, len(a.Value.Group())),
		}
	}

	for _, attr := range a.Value.Group() {
		parseValue(attr, groupAttr)
	}

	if isSubgroup && len(groupAttr.Children) != 0 {
		parent.Children = append(parent.Children, groupAttr)
	}
}

func prune(a *Attr) {
	for i, child := range a.Children {
		if child.empty() {
			a.Children = slices.Delete(a.Children, i, i+1)
			continue
		}

		prune(child)
	}
}
