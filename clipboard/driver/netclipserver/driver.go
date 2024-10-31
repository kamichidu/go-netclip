package driver

import (
	"context"
	"sync"
	"time"

	"github.com/kamichidu/go-netclip/clipboard"
	"github.com/kamichidu/go-netclip/config"
	"github.com/kamichidu/go-netclip/netclippb"
	"google.golang.org/grpc"
)

type drv struct {
	URL string

	once sync.Once

	netclipClient netclippb.NetclipClient
}

func newDriver(cfg *config.NetclipConfig) (clipboard.Store, error) {
	urlStr, _ := cfg.Get("netclip.server.url").(string)
	return &drv{
		URL: urlStr,
	}, nil
}

func (d *drv) Init(ctx context.Context) error {
	var retErr error
	d.once.Do(func() {
		var opts []grpc.DialOption
		opts = append(opts, grpc.WithInsecure())
		conn, err := grpc.NewClient(d.URL, opts...)
		if err != nil {
			retErr = err
			return
		}
		d.netclipClient = netclippb.NewNetclipClient(conn)
	})
	return retErr
}

func (d *drv) List(ctx context.Context) ([]*netclippb.Container, error) {
	if err := d.Init(ctx); err != nil {
		return nil, err
	}
	r, err := d.netclipClient.List(ctx, &netclippb.ListRequest{})
	if err != nil {
		return nil, err
	}
	return r.Items, nil
}

func (d *drv) Copy(ctx context.Context, v string) error {
	if err := d.Init(ctx); err != nil {
		return err
	}
	_, err := d.netclipClient.Copy(ctx, &netclippb.CopyRequest{
		Value: v,
	})
	return err
}

func (d *drv) Paste(ctx context.Context) (*netclippb.Container, error) {
	if err := d.Init(ctx); err != nil {
		return nil, err
	}
	r, err := d.netclipClient.Paste(ctx, &netclippb.PasteRequest{})
	if err != nil {
		return nil, err
	}
	return r.Value, nil
}

func (d *drv) Remove(ctx context.Context, timestamps ...time.Time) error {
	if len(timestamps) == 0 {
		return nil
	}
	if err := d.Init(ctx); err != nil {
		return err
	}
	var req netclippb.RemoveRequest
	req.Timestamps = make([]int64, len(timestamps))
	for i := range timestamps {
		req.Timestamps[i] = timestamps[i].Unix()
	}
	_, err := d.netclipClient.Remove(ctx, &req)
	return err
}

func (d *drv) Expire(ctx context.Context, t time.Time) error {
	if err := d.Init(ctx); err != nil {
		return err
	}
	_, err := d.netclipClient.Expire(ctx, &netclippb.ExpireRequest{
		ExpiresAt: t.Unix(),
	})
	return err
}

func (d *drv) Watch(ctx context.Context) <-chan clipboard.Event {
	ch := make(chan clipboard.Event)
	go func() {
		defer close(ch)

		if err := d.Init(ctx); err != nil {
			ch <- clipboard.Event{
				Err: err,
			}
			return
		}

		stream, err := d.netclipClient.Watch(ctx, &netclippb.WatchRequest{})
		if err != nil {
			ch <- clipboard.Event{
				Err: err,
			}
			return
		}
		for {
			res, err := stream.Recv()
			if err != nil {
				ch <- clipboard.Event{
					Err: err,
				}
				return
			}
			ch <- clipboard.Event{
				Type:  clipboard.EventCopy,
				Value: res.Value,
			}
		}
	}()
	return ch
}

func init() {
	clipboard.Register("netclip.server", newDriver)
	config.Register("netclip.server.url", config.NewSpec("", config.TypeString))
}
