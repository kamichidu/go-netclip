package clipboard

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/kamichidu/go-netclip/config"
	"github.com/kamichidu/go-netclip/netclippb"
)

type NewStoreFunc func(*config.NetclipConfig) (Store, error)

type Store interface {
	List(context.Context) ([]*netclippb.Container, error)
	Copy(context.Context, string) error
	Paste(context.Context) (*netclippb.Container, error)
	Remove(context.Context, ...time.Time) error
	Expire(context.Context, time.Time) error
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

func NewStore(driverName string, cfg *config.NetclipConfig) (Store, error) {
	newStore, ok := Lookup(driverName)
	if !ok {
		return nil, errors.New("invalid store driver: " + driverName)
	}
	return newStore(cfg)
}
