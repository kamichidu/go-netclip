package clipboard

import (
	"encoding/json"
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

func (v Event) MarshalJSON() ([]byte, error) {
	m := map[string]any{
		"type": v.Type.String(),
	}
	if v.Value != "" {
		m["value"] = v.Value
	}
	if v.Err != nil {
		m["err"] = v.Err.Error()
	}
	return json.Marshal(m)
}

func (v Event) String() string {
	return fmt.Sprintf("%v - err=%v value=%v", v.Type, v.Err, Shorten(v.Value))
}

func Shorten(s string) string {
	const maxLen = 80

	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\n", "\\n")
	chars := []rune(s)
	if len(chars) > maxLen {
		chars = append(chars[:maxLen-3], []rune("...")...)
	}
	return string(chars)
}
