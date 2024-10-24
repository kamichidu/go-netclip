package driver

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"github.com/kamichidu/go-netclip/clipboard"
	"go.uber.org/multierr"
	"google.golang.org/api/option"
)

type drv struct {
	ProjectID string

	Database string

	CredentialsFile string

	once sync.Once

	firestoreClient *firestore.Client
}

func newDriver(m map[string]any) (clipboard.Store, error) {
	projectID, _ := m["projectId"].(string)
	database, _ := m["database"].(string)
	credFile, _ := m["credentials"].(string)
	return &drv{
		ProjectID:       projectID,
		Database:        database,
		CredentialsFile: credFile,
	}, nil
}

func (d *drv) Init(ctx context.Context) error {
	var retErr error
	d.once.Do(func() {
		newFirestoreClient := func() (*firestore.Client, error) {
			var opts []option.ClientOption
			if d.CredentialsFile != "" {
				opts = append(opts, option.WithCredentialsFile(d.CredentialsFile))
			}
			if d.Database == "" {
				return firestore.NewClient(ctx, d.ProjectID, opts...)
			} else {
				return firestore.NewClientWithDatabase(ctx, d.ProjectID, d.Database, opts...)
			}
		}
		if c, err := newFirestoreClient(); err != nil {
			retErr = multierr.Append(retErr, err)
		} else {
			d.firestoreClient = c
		}
	})
	return retErr
}

func (d *drv) List(ctx context.Context) ([]*clipboard.Container, error) {
	if err := d.Init(ctx); err != nil {
		return nil, err
	}

	l, err := d.firestoreClient.Collection("clipboard").
		OrderBy("timestamp", firestore.Desc).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	out := make([]*clipboard.Container, len(l))
	for i := range l {
		out[i] = newContainerFromDocumentSnapshot(l[i])
	}
	return out, nil
}

func (d *drv) Copy(ctx context.Context, v string) error {
	if err := d.Init(ctx); err != nil {
		return err
	}

	value := &clipboard.Container{
		Value:     v,
		Timestamp: time.Now().Truncate(time.Second),
	}
	l, err := d.firestoreClient.Collection("clipboard").
		Where("md5", "==", value.MD5()).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	}
	for i := range l {
		other := newContainerFromDocumentSnapshot(l[i])
		if other.Value != value.Value {
			continue
		}
		if _, err := l[i].Ref.Delete(ctx, firestore.LastUpdateTime(l[i].UpdateTime)); err != nil {
			return err
		}
	}
	doc := d.firestoreClient.Collection("clipboard").Doc(uuid.New().String())
	if _, err := doc.Create(ctx, asMapFromContainer(value)); err != nil {
		return err
	}
	return nil
}

func (d *drv) Paste(ctx context.Context) (string, error) {
	if err := d.Init(ctx); err != nil {
		return "", err
	}

	itr := d.firestoreClient.Collection("clipboard").Query.
		OrderBy("timestamp", firestore.Desc).
		Limit(1).
		Documents(ctx)
	l, err := itr.GetAll()
	if err != nil {
		return "", err
	}
	switch n := len(l); n {
	case 0:
		return "", nil
	case 1:
		// ok
	default:
		panic(fmt.Sprintf("invalid number of documents: %d", n))
	}
	v := newContainerFromDocumentSnapshot(l[0])
	return v.Value, nil
}

func (d *drv) Remove(ctx context.Context, timestamps ...time.Time) error {
	if len(timestamps) == 0 {
		return nil
	}
	if err := d.Init(ctx); err != nil {
		return err
	}

	for i := range timestamps {
		timestamps[i] = timestamps[i].Truncate(time.Second)
	}
	l, err := d.firestoreClient.Collection("clipboard").
		Where("timestamp", "in", timestamps).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	} else if len(l) != len(timestamps) {
		return fmt.Errorf("invalid number of documents found, expects %d but got %d", len(timestamps), len(l))
	}
	return d.firestoreClient.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		for i := range l {
			if err := tx.Delete(l[i].Ref, firestore.LastUpdateTime(l[i].UpdateTime)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *drv) Expiry(ctx context.Context, duration time.Duration) error {
	if err := d.Init(ctx); err != nil {
		return err
	}

	expiry := time.Now().Add(duration)
	l, err := d.firestoreClient.Collection("clipboard").
		Where("timestamp", "<=", expiry).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	}
	if len(l) == 0 {
		return nil
	}
	return d.firestoreClient.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		for i := range l {
			if err := tx.Delete(l[i].Ref, firestore.LastUpdateTime(l[i].UpdateTime)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *drv) Watch(ctx context.Context) <-chan clipboard.Event {
	ch := make(chan clipboard.Event)
	go func() {
		defer close(ch)

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		if err := d.Init(ctx); err != nil {
			ch <- clipboard.Event{
				Err: err,
			}
			return
		}

		itr := d.firestoreClient.Collection("clipboard").Query.
			OrderBy("timestamp", firestore.Asc).
			StartAfter(time.Now()).
			Snapshots(ctx)
		go func() {
			defer itr.Stop()
			<-ctx.Done()
		}()
		for {
			qs, err := itr.Next()
			if err != nil {
				ch <- clipboard.Event{
					Err: err,
				}
				return
			}
			for _, dc := range qs.Changes {
				var evt clipboard.Event
				switch dc.Kind {
				case firestore.DocumentAdded:
					evt.Type = clipboard.EventCopy
				case firestore.DocumentModified:
					evt.Type = clipboard.EventCopy
				case firestore.DocumentRemoved:
					evt.Type = clipboard.EventRemove
				}
				v := newContainerFromDocumentSnapshot(dc.Doc)
				evt.Value = v.Value
				ch <- evt
			}
		}
	}()
	return ch
}

func newContainerFromDocumentSnapshot(src *firestore.DocumentSnapshot) *clipboard.Container {
	var out clipboard.Container
	m := src.Data()
	if v, ok := m["value"].(string); ok {
		out.Value = v
	}
	if v, ok := m["timestamp"].(time.Time); ok {
		out.Timestamp = v
	}
	return &out
}

func asMapFromContainer(src *clipboard.Container) map[string]any {
	return map[string]any{
		"value":     src.Value,
		"md5":       src.MD5(),
		"timestamp": src.Timestamp.Truncate(time.Second),
	}
}

func init() {
	clipboard.Register("firestore", newDriver)
}
