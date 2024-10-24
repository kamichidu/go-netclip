package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
)

const (
	KeyServerURL = "server.url"
)

var (
	configSpec = map[string]spec{
		KeyServerURL: spec{
			Default: "http://localhost:30564",
			Types: []reflect.Type{
				reflect.TypeOf(""),
			},
		},
		"firestore.projectId": spec{
			Default: "",
			Types: []reflect.Type{
				reflect.TypeOf(""),
			},
		},
		"firestore.database": spec{
			Default: "",
			Types: []reflect.Type{
				reflect.TypeOf(""),
			},
		},
		"firestore.credentials": spec{
			Default: "",
			Types: []reflect.Type{
				reflect.TypeOf(""),
			},
		},
	}
)

type spec struct {
	Default any

	Types []reflect.Type
}

type NetclipConfig struct {
	Path string

	data map[string]json.RawMessage
}

func NewNetclipConfigFromFile(name string) (*NetclipConfig, error) {
	var v NetclipConfig
	v.Path = name
	v.data = map[string]json.RawMessage{}

	b, err := os.ReadFile(name)
	if errors.Is(err, os.ErrNotExist) {
		return &v, nil
	} else if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &v.data); err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *NetclipConfig) Keys() []string {
	l := make([]string, 0, len(configSpec))
	for k := range configSpec {
		l = append(l, k)
	}
	sort.Strings(l)
	return l
}

func (c *NetclipConfig) Set(key string, value any) {
	spec, ok := configSpec[key]
	if !ok {
		panic("invalid config key: " + key)
	}
	if !validateType(value, spec.Types) {
		panic(fmt.Sprintf("invalid value type: %T", value))
	}
	v, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	c.data[key] = v
}

func (c *NetclipConfig) Get(key string) any {
	spec, ok := configSpec[key]
	if !ok {
		panic("invalid config key: " + key)
	}
	value, ok := c.data[key]
	if !ok {
		return spec.Default
	}
	var v any
	if err := json.Unmarshal(value, &v); err != nil {
		panic(err)
	}
	if !validateType(v, spec.Types) {
		panic(fmt.Sprintf("invalid value type: %T", v))
	}
	if v == nil {
		return spec.Default
	}
	return v
}

func (c *NetclipConfig) Commit() error {
	return c.Write(c.Path)
}

func (c *NetclipConfig) Write(name string) error {
	dir := filepath.Dir(name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		panic(err)
	}
	data = append(data, []byte("\n")...)
	return os.WriteFile(name, data, 0644)
}

func validateType(v any, typs []reflect.Type) bool {
	if v == nil {
		return true
	}
	typ := reflect.TypeOf(v)
	for i := range typs {
		if typ == typs[i] {
			return true
		}
	}
	return false
}
