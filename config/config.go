package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

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
	return reg.Names()
}

func (c *NetclipConfig) ValidKey(key string) bool {
	_, ok := reg.Lookup(key)
	return ok
}

func (c *NetclipConfig) Set(key string, value any) {
	spec, ok := reg.Lookup(key)
	if !ok {
		panic("invalid config key: " + key)
	}
	if !spec.Validate(value) {
		panic(fmt.Sprintf("invalid value type: %T", value))
	}
	v, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	c.data[key] = v
}

func (c *NetclipConfig) Get(key string) any {
	spec, ok := reg.Lookup(key)
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
	if !spec.Validate(v) {
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
