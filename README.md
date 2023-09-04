# EasySlog

EasySlog makes it easy to write custom formatters for slog by implementing the plumbing of the `slog.Handler` and exposes a simple interface to implement your own formatter.

The JSON formatter implemented in `easyslog_test.go` passes the `slogtest.TestHandler` tests and is a good example of how to implement a formatter.

## Usage

```go
type myFormatter struct{}

func (f myFormatter) Format(w io.Writer, rec easyslog.Record) error {
	attrs := make([]string, 0, len(rec.Attrs))
	for _, attr := range rec.Attrs {
		key := strings.Join(attr.Keys, ".")
		attrs = append(attrs, fmt.Sprintf("%s=%s", key, attr.Value.String()))
	}

	w.Write([]byte(fmt.Sprintf("%s %s %s %s\n", rec.Time, rec.Level, rec.Message, strings.Join(attrs, " "))))
	return nil
}

logger := slog.New(easyslog.New(os.Stdout, myFormatter{}, nil))
logger.Debug("hello world", slog.String("foo", "bar"))
```

See also the `prettylog` package for a more complete example.
