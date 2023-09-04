package prettylog

import (
	"io"
	"log/slog"
	"strings"

	"github.com/blakewilliams/easyslog"
	"github.com/fatih/color"
)

// Formatter implements easyslog.Formatter and can be used to render "pretty"
// slog logs.
type Formatter struct {
	// Determines if color is used or not
	NoColor bool
}

var _ easyslog.Formatter = (*Formatter)(nil)

// Levels maps a level to a specific prefix to log. Levels not in this list will
// render as unknown `[UNK]`
var Levels = map[slog.Level]string{
	slog.LevelDebug: "[DBG]",
	slog.LevelInfo:  "[INF]",
	slog.LevelWarn:  "[WRN]",
	slog.LevelError: "[ERR]",
}

// LevelColors maps log levels to colors when color is enabled. Levels not in
// this list will render as cyan.
var LevelColors = map[slog.Level]color.Attribute{
	slog.LevelDebug: color.FgGreen,
	slog.LevelInfo:  color.FgBlue,
	slog.LevelWarn:  color.FgYellow,
	slog.LevelError: color.FgRed,
}

func (f Formatter) Format(w io.Writer, record easyslog.Record) error {
	colorAttr := color.FgCyan
	if attr, ok := LevelColors[record.Level]; ok {
		colorAttr = attr
	}
	c := color.New(colorAttr)

	if f.NoColor {
		c.DisableColor()
	}

	level := "[UNK]"
	if definedLevel, ok := Levels[record.Level]; ok {
		level = definedLevel
	}

	c.Add(color.Bold).Fprint(w, level)
	_, _ = w.Write([]byte(" "))
	_, _ = w.Write([]byte(record.Message))
	_, _ = w.Write([]byte(" "))

	for i, attr := range record.Attrs {
		key := strings.Join(attr.Keys, ".")
		c.Fprint(w, key)
		_, _ = w.Write([]byte("="))
		_, _ = w.Write([]byte(attr.Value.String()))
		if i != len(record.Attrs)-1 {
			_, _ = w.Write([]byte(" "))
		}
	}

	return nil
}
