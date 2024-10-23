package clipboard

import (
	"context"
	"errors"
	"sync"
	"time"
)

type NewStoreFunc func(map[string]any) (Store, error)

type Store interface {
	List(context.Context) ([]*Container, error)
	Copy(context.Context, string) error
	Paste(context.Context) (string, error)
	Remove(context.Context, ...time.Time) error
	Expiry(context.Context, time.Duration) error
	Watch(context.Context) <-chan Event
}

var (
	drivers   = map[string]NewStoreFunc{}
	driversMu sync.Mutex
)

func Register(name string, fn NewStoreFunc) {
	driversMu.Lock()
	defer driversMu.Unlock()

	if _, dup := drivers[name]; dup {
		panic("imported twice: %v" + name)
	}
	drivers[name] = fn
}

func Lookup(name string) (NewStoreFunc, bool) {
	driversMu.Lock()
	defer driversMu.Unlock()

	fn, ok := drivers[name]
	return fn, ok
}

func NewStore(driverName string, args map[string]any) (Store, error) {
	newStore, ok := Lookup(driverName)
	if !ok {
		return nil, errors.New("invalid store driver: " + driverName)
	}
	return newStore(args)
}
