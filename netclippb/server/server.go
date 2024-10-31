package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/kamichidu/go-netclip/clipboard"
	"github.com/kamichidu/go-netclip/netclippb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

type Netclip struct {
	netclippb.UnsafeNetclipServer

	stateDir string
}

func (srv *Netclip) List(ctx context.Context, req *netclippb.ListRequest) (*netclippb.ListResponse, error) {
	files, err := os.ReadDir(srv.stateDir)
	if errors.Is(err, os.ErrNotExist) {
		return &netclippb.ListResponse{}, nil
	} else if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	var out netclippb.ListResponse
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}
		v, err := readContainer(filepath.Join(srv.stateDir, file.Name()))
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		out.Items = append(out.Items, v)
	}
	slices.SortFunc(out.Items, func(a, b *netclippb.Container) int {
		return int(b.Timestamp - a.Timestamp)
	})
	return &out, nil
}

func (srv *Netclip) Copy(ctx context.Context, req *netclippb.CopyRequest) (*netclippb.CopyResponse, error) {
	var v netclippb.Container
	v.Value = req.Value
	v.Md5 = clipboard.MD5(req.Value)
	v.Timestamp = time.Now().Unix()

	if err := writeContainer(filepath.Join(srv.stateDir, v.Md5+".json"), &v); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &netclippb.CopyResponse{}, nil
}

func (srv *Netclip) Paste(ctx context.Context, req *netclippb.PasteRequest) (*netclippb.PasteResponse, error) {
	listRes, err := srv.List(ctx, &netclippb.ListRequest{})
	if err != nil {
		return nil, err
	}
	items := listRes.Items
	if len(items) == 0 {
		return &netclippb.PasteResponse{}, nil
	}
	latest := slices.MaxFunc(listRes.Items, func(a, b *netclippb.Container) int {
		return int(a.Timestamp - b.Timestamp)
	})
	return &netclippb.PasteResponse{
		Value: latest,
	}, nil
}

func (srv *Netclip) Remove(ctx context.Context, req *netclippb.RemoveRequest) (*netclippb.RemoveResponse, error) {
	if len(req.Timestamps) == 0 {
		return &netclippb.RemoveResponse{}, nil
	}

	listRes, err := srv.List(ctx, &netclippb.ListRequest{})
	if err != nil {
		return nil, err
	}
	items := listRes.Items
	if len(items) == 0 {
		return &netclippb.RemoveResponse{}, nil
	}
	var removals []*netclippb.Container
	for _, ts := range req.Timestamps {
		idx := slices.IndexFunc(items, func(v *netclippb.Container) bool {
			return v.Timestamp == ts
		})
		if idx < 0 {
			return nil, status.Errorf(codes.NotFound, "entity not found from timestamp: %v", ts)
		}
		removals = append(removals, items[idx])
	}
	for _, v := range removals {
		if err := os.Remove(filepath.Join(srv.stateDir, v.Md5+".json")); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	return &netclippb.RemoveResponse{}, nil
}

func (srv *Netclip) Expire(ctx context.Context, req *netclippb.ExpireRequest) (*netclippb.ExpireResponse, error) {
	listRes, err := srv.List(ctx, &netclippb.ListRequest{})
	if err != nil {
		return nil, err
	}
	items := listRes.Items
	if len(items) == 0 {
		return &netclippb.ExpireResponse{}, nil
	}

	var removals []*netclippb.Container
	for _, v := range items {
		if v.Timestamp > req.ExpiresAt {
			continue
		}
		removals = append(removals, v)
	}
	for _, v := range removals {
		if err := os.Remove(filepath.Join(srv.stateDir, v.Md5+".json")); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	return &netclippb.ExpireResponse{}, nil
}

func (srv *Netclip) Watch(req *netclippb.WatchRequest, stream netclippb.Netclip_WatchServer) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	defer watcher.Close()

	ready := make(chan struct{})
	go func() {
		defer close(ready)

		if err := watcher.Add(srv.stateDir); err != nil {
			panic(err)
		}
	}()
	<-ready

	fn := func(evt fsnotify.Event) error {
		if !evt.Has(fsnotify.Create) {
			return nil
		}
		v, err := readContainer(evt.Name)
		if err != nil {
			return err
		}
		return stream.Send(&netclippb.WatchResponse{
			Value: v,
		})
	}

	for {
		select {
		case evt := <-watcher.Events:
			if err := fn(evt); err != nil {
				log.Printf("error: %v", err)
			}
		case err := <-watcher.Errors:
			log.Printf("error: %v", err)
		case <-stream.Context().Done():
			err := stream.Context().Err()
			if errors.Is(err, context.Canceled) {
				return status.Error(codes.Canceled, err.Error())
			} else {
				return status.Error(codes.DeadlineExceeded, err.Error())
			}
		}
	}
}

func readContainer(name string) (*netclippb.Container, error) {
	b, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	var v netclippb.Container
	if err := protojson.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func writeContainer(name string, v *netclippb.Container) error {
	tmp, err := os.CreateTemp("", "")
	if err != nil {
		return err
	}
	defer tmp.Close()

	var m protojson.MarshalOptions
	m.Indent = "  "
	m.Multiline = true
	b, err := m.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := io.Copy(tmp, bytes.NewReader(b)); err != nil {
		return err
	}
	if _, err := io.WriteString(tmp, "\n"); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), name); err != nil {
		return err
	}
	return nil
}

type RunConfig struct {
	Addr string

	StateDir string
}

func Run(cfg *RunConfig) error {
	var service Netclip
	service.stateDir = cfg.StateDir

	srv := grpc.NewServer()
	netclippb.RegisterNetclipServer(srv, &service)

	lis, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return err
	}
	defer lis.Close()

	return srv.Serve(lis)
}
