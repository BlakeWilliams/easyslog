package easyslog

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"testing/slogtest"
)

// Implement a JSON formatter to use with slogtest.TestHandler
type JSONFormatter struct{}

var _ Formatter = (*JSONFormatter)(nil)

func (formatter JSONFormatter) Format(w io.Writer, record Record) error {
	result := map[string]any{
		"msg":   record.Message,
		"level": record.Level.String(),
	}

	if !record.Time.IsZero() {
		result["time"] = record.Time
	}

	for _, attr := range record.Attrs {
		writeTo := result
		for _, key := range attr.Keys[:len(attr.Keys)-1] {
			if _, ok := writeTo[key]; !ok {
				writeTo[key] = map[string]any{}
			}

			writeTo = writeTo[key].(map[string]any)
		}

		writeTo[attr.Keys[len(attr.Keys)-1]] = attr.Value.String()
	}

	toWrite, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, _ = w.Write(toWrite)
	return nil
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

	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkEasySlog(b *testing.B) {
	formatter := JSONFormatter{}
	handler := New(io.Discard, formatter, &Options{Level: slog.LevelDebug})

	l := slog.New(handler)

	for i := 0; i < b.N; i++ {
		l.Info("hello")
		l.With("foo", "bar").WithGroup("X-Files").With("Fox", "Mulder", "Dana", "Scully").Info("The truth is out there", "spooky", true)
	}
}
