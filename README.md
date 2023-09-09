# EasySlog

EasySlog makes it easy to write custom formatters for slog by implementing the plumbing of the `slog.Handler` and exposes a simple interface to implement your own formatter. This is less performant than writing a custom slog formatter, but it is much easier to write and test.

The JSON formatter implemented in `easyslog_test.go` passes the `slogtest.TestHandler` tests and is a good example of how to implement a formatter.

## Usage

To use EasySlog you need to implement the `easyslog.Formatter` interface and pass it to `easyslog.NewHandler`:

```go
Formatter interface {
    Format(w io.Writer, r Record) error
}
```

Here's an example of implementing a formatter:

```go
type myFormatter struct{}

func (f myFormatter) Format(w io.Writer, rec easyslog.Record) error {
    attrs := make([]string, 0)
    for _, attr := range rec.Attrs {
        attrs = append(attrs, formatAttr(attr)...)
    }

    w.Write([]byte(fmt.Sprintf("%s %s %s %s\n", rec.Time, rec.Level, rec.Message, strings.Join(attrs, " "))))
    return nil
}

func formatAttr(attr easyslog.Attr) []string {
    if attr.IsGroup() {
        attrs := make([]string, 0, len(attr.Children))

        for _, child := range attr.Children {
            attrs = append(attrs, formatAttr(child)...)
        }
        return attrs
    }

    return []string{fmt.Sprintf("%s=%s", attr.Key, attr.Value.String())}
}

// Usage:
// slog.New(easyslog.NewHandler(myFormatter{}, nil))
```

See also the `prettylog` package for a more complete example.
