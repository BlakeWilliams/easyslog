package prettylog

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/blakewilliams/easyslog"
	"github.com/fatih/color"
	"github.com/stretchr/testify/require"
)

func TestFormat(t *testing.T) {
	var buf bytes.Buffer
	handler := easyslog.New(&buf, Formatter{}, nil)
	l := slog.New(handler)

	l.Info("omg", "foo", "bar", "baz", "quux")

	require.Equal(t, "[INF] omg foo=bar baz=quux \n", buf.String())
}

func TestColorDisabled(t *testing.T) {
	defer func() {
		color.NoColor = true
	}()
	color.NoColor = false

	var buf bytes.Buffer
	handler := easyslog.New(&buf, Formatter{NoColor: true}, nil)
	l := slog.New(handler)

	l.Info("omg", "foo", "bar", "baz", "quux")

	require.Equal(t, "[INF] omg foo=bar baz=quux \n", buf.String())
}

func TestUnknownLogLevels(t *testing.T) {
	var buf bytes.Buffer
	handler := easyslog.New(&buf, Formatter{}, nil)
	l := slog.New(handler)

	l.Log(context.Background(), 7, "omg", "foo", "bar", "baz", "quux")

	require.Equal(t, "[UNK] omg foo=bar baz=quux \n", buf.String())
}

func TestGroups(t *testing.T) {
	var buf bytes.Buffer
	handler := easyslog.New(&buf, Formatter{}, nil)
	l := slog.New(handler)

	l.Info("msg", slog.Group("request", "method", "get", "path", "/"))

	require.Equal(t, "[INF] msg request.method=get request.path=/ \n", buf.String())
}
