package easyslog

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"testing/slogtest"

	"github.com/stretchr/testify/require"
)

// Implement a JSON formatter to use with slogtest.TestHandler
type JSONFormatter struct{}

var _ Formatter = (*JSONFormatter)(nil)

func (formatter JSONFormatter) Format(w io.Writer, record Record) error {
	result := make(map[string]any, len(record.Attrs))
	result["msg"] = record.Message
	result["level"] = record.Level.String()

	if !record.Time.IsZero() {
		result["time"] = record.Time
	}

	for _, attr := range record.Attrs {
		writeAttr(result, attr)
	}

	toWrite, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, _ = w.Write(toWrite)
	return nil
}

func writeAttr(dst map[string]any, attr *Attr) {
	if attr.Children == nil || len(attr.Children) == 0 {
		dst[attr.Key] = attr.Value.String()
		return
	}

	dst[attr.Key] = make(map[string]any)
	for _, child := range attr.Children {
		writeAttr(dst[attr.Key].(map[string]any), child)
	}
}

func TestEasySlog(t *testing.T) {
	formatter := JSONFormatter{}
	var b bytes.Buffer
	handler := New(&b, formatter, &Options{Level: slog.LevelDebug})

	err := slogtest.TestHandler(handler, func() []map[string]any {
		var results []map[string]any
		for _, line := range bytes.Split(b.Bytes(), []byte{'\n'}) {
			if len(line) == 0 {
				continue
			}

			var result map[string]any
			if err := json.Unmarshal(line, &result); err != nil {
				t.Fatal(err)
			}
			results = append(results, result)
		}

		return results
	})

	require.NoError(t, err)
}

type FastJSONFormatter struct{}

var _ Formatter = (*FastJSONFormatter)(nil)

func (formatter FastJSONFormatter) Format(w io.Writer, record Record) error {
	result := make(map[string]any, len(record.Attrs))
	result["msg"] = record.Message
	result["level"] = record.Level.String()

	if !record.Time.IsZero() {
		result["time"] = record.Time
	}

	var buf bytes.Buffer
	buf.Write([]byte("{"))

	buf.Write([]byte("\"msg\":\""))
	buf.Write([]byte(record.Message))
	buf.Write([]byte("\""))

	buf.Write([]byte(",\"level\":\""))
	buf.Write([]byte(record.Level.String()))
	buf.Write([]byte("\""))

	if !record.Time.IsZero() {
		buf.Write([]byte(",\"time\":\""))
		buf.Write([]byte(record.Time.String()))
		buf.Write([]byte("\""))
	}

	for _, attr := range record.Attrs {
		fastWriteAttr(&buf, attr, false)
	}

	buf.Write([]byte("}"))
	_, _ = w.Write(buf.Bytes())
	return nil
}

func fastWriteAttr(buf *bytes.Buffer, attr *Attr, first bool) {
	if !first {
		buf.Write([]byte(","))
	}
	if attr.Children == nil || len(attr.Children) == 0 {
		buf.Write([]byte(`"`))
		buf.Write([]byte(attr.Key))
		buf.Write([]byte(`":`))

		switch attr.Value.Kind() {
		case slog.KindInt64, slog.KindFloat64, slog.KindUint64:
			buf.Write([]byte(attr.Value.String()))
		default:
			buf.Write([]byte("\""))
			buf.Write([]byte(attr.Value.String()))
			buf.Write([]byte("\""))
		}
		return
	}

	buf.Write([]byte(`"`))
	buf.Write([]byte(attr.Key))
	buf.Write([]byte(`":`))

	buf.Write([]byte("{"))
	for i, child := range attr.Children {
		fastWriteAttr(buf, child, i == 0)
	}
	buf.Write([]byte("}"))
}

func BenchmarkEasySlog(b *testing.B) {
	formatter := FastJSONFormatter{}
	handler := New(io.Discard, formatter, &Options{Level: slog.LevelDebug})

	l := slog.New(handler)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l.Info("hello")
		l.With("foo", "bar").WithGroup("X-Files").With("Fox", "Mulder", "Dana", "Scully").Info("The truth is out there", "spooky", true)
	}
}

func BenchmarkSlog(b *testing.B) {
	l := slog.New(slog.NewJSONHandler(io.Discard, nil))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l.Info("hello")
		l.With("foo", "bar").WithGroup("X-Files").With("Fox", "Mulder", "Dana", "Scully").Info("The truth is out there", "spooky", true)
	}
}
