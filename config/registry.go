package config

import (
	"sort"
	"sync"
)

var (
	reg = &registry{}
)

func Register(name string, spec Spec) {
	reg.Register(name, spec)
}

type registry struct {
	data map[string]Spec

	mu sync.Mutex
}

func (r *registry) Register(name string, spec Spec) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, dup := r.data[name]; dup {
		panic("already exists, imported twice? " + name)
	}
	if r.data == nil {
		r.data = map[string]Spec{}
	}
	r.data[name] = spec
}

func (r *registry) Lookup(name string) (Spec, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	v, ok := r.data[name]
	return v, ok
}

func (r *registry) Names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	l := make([]string, 0, len(r.data))
	for k := range r.data {
		l = append(l, k)
	}
	sort.Strings(l)
	return l
}
