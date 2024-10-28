package config

import "reflect"

var (
	TypeString = reflect.TypeOf("")
	TypeInt64  = reflect.TypeOf(int64(0))
	TypeBool   = reflect.TypeOf(false)
)

type Spec struct {
	Default any

	Types []reflect.Type
}

func NewSpec(default_ any, types ...reflect.Type) Spec {
	return Spec{
		Default: default_,
		Types:   types,
	}
}

func (s *Spec) Validate(v any) bool {
	if v == nil {
		return true
	}
	typ := reflect.TypeOf(v)
	for i := range s.Types {
		if typ == s.Types[i] {
			return true
		}
	}
	return false
}
