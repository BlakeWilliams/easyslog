package easyslog

import (
	"log/slog"
)

type (
	// Attr is a slog attribute that contains the key, the value, and a map of
	// child attributes. Attr values that return true for `IsGroup` should not
	// use the `Value` field and vice-versa.
	Attr struct {
		// The key of the attribute.
		Key string
		// The underlying slog.Value attribute being logged. `Resolve` is
		// already called on values that implement `slog.LogValuer``.
		Value slog.Value
		// Children holds pointers to each of the nested attributes if they exist.
		Children []*Attr
	}
)

// Clone the existing tree for use in the formatter
func (a *Attr) clone() *Attr {
	attr := &Attr{
		Key:      a.Key,
		Value:    a.Value,
		Children: make([]*Attr, len(a.Children)),
	}

	for i, child := range a.Children {
		attr.Children[i] = child.clone()
	}

	return attr
}

// Returns true if this is a dead-end node and should not be rendered
func (a *Attr) empty() bool {
	return a.Value.Any() == nil && (a.Children == nil || len(a.Children) == 0)
}

// Returns true if this Attr represents a group and its Value field should be
// ignored.
func (a *Attr) IsGroup() bool {
	return len(a.Children) > 0
}
