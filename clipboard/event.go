package clipboard

import (
	"fmt"
	"strings"
)

type EventType int

const (
	EventCopy EventType = iota
	EventRemove
)

func (v EventType) String() string {
	switch v {
	case EventCopy:
		return "copy"
	case EventRemove:
		return "remove"
	default:
		panic(fmt.Sprintf("invalid EventType(%d)", int(v)))
	}
}

type Event struct {
	Type EventType

	Value string

	Err error
}

func (v Event) String() string {
	const maxLen = 80

	s := v.Value
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\n", "\\n")
	chars := []rune(s)
	if len(chars) > maxLen {
		chars = append(chars[:maxLen-3], []rune("...")...)
	}
	return fmt.Sprintf("%v - err=%v value=%v", v.Type, v.Err, string(chars))
}
