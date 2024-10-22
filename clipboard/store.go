package clipboard

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"go.uber.org/multierr"
)

type Store struct {
	ProjectID string

	Database string

	once sync.Once

	firestoreClient *firestore.Client
}

func (s *Store) Init(ctx context.Context) error {
	var retErr error
	s.once.Do(func() {
		newFirestoreClient := func() (*firestore.Client, error) {
			if s.Database == "" {
				return firestore.NewClient(ctx, s.ProjectID)
			} else {
				return firestore.NewClientWithDatabase(ctx, s.ProjectID, s.Database)
			}
		}
		if c, err := newFirestoreClient(); err != nil {
			retErr = multierr.Append(retErr, err)
		} else {
			s.firestoreClient = c
		}
	})
	return retErr
}

func (s *Store) List(ctx context.Context) ([]*Container, error) {
	if err := s.Init(ctx); err != nil {
		return nil, err
	}

	l, err := s.firestoreClient.Collection("clipboard").
		OrderBy("timestamp", firestore.Desc).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	out := make([]*Container, len(l))
	for i := range l {
		var v Container
		if err := l[i].DataTo(&v); err != nil {
			return nil, err
		}
		out[i] = &v
	}
	return out, nil
}

func (s *Store) Copy(ctx context.Context, v string) error {
	if err := s.Init(ctx); err != nil {
		return err
	}

	value := &Container{
		Value:     v,
		MD5:       md5sum(v),
		Timestamp: time.Now().Truncate(time.Second),
	}
	l, err := s.firestoreClient.Collection("clipboard").
		Where("md5", "==", value.MD5).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	}
	for i := range l {
		var other Container
		if err := l[i].DataTo(&other); err != nil {
			return err
		}
		if other.Value != value.Value {
			continue
		}
		if _, err := l[i].Ref.Delete(ctx, firestore.LastUpdateTime(l[i].UpdateTime)); err != nil {
			return err
		}
	}
	doc := s.firestoreClient.Collection("clipboard").Doc(uuid.New().String())
	if _, err := doc.Create(ctx, &value); err != nil {
		return err
	}
	return nil
}

func (s *Store) Paste(ctx context.Context) (string, error) {
	if err := s.Init(ctx); err != nil {
		return "", err
	}

	itr := s.firestoreClient.Collection("clipboard").Query.
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
	var v Container
	if err := l[0].DataTo(&v); err != nil {
		return "", err
	}
	return v.Value, nil
}

func (s *Store) Remove(ctx context.Context, timestamps ...time.Time) error {
	if len(timestamps) == 0 {
		return nil
	}
	if err := s.Init(ctx); err != nil {
		return err
	}

	for i := range timestamps {
		timestamps[i] = timestamps[i].Truncate(time.Second)
	}
	l, err := s.firestoreClient.Collection("clipboard").
		Where("timestamp", "in", timestamps).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	} else if len(l) != len(timestamps) {
		return fmt.Errorf("invalid number of documents found, expects %d but got %d", len(timestamps), len(l))
	}
	return s.firestoreClient.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		for i := range l {
			if err := tx.Delete(l[i].Ref, firestore.LastUpdateTime(l[i].UpdateTime)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) Expiry(ctx context.Context, d time.Duration) error {
	if err := s.Init(ctx); err != nil {
		return err
	}

	expiry := time.Now().Add(d)
	l, err := s.firestoreClient.Collection("clipboard").
		Where("timestamp", "<=", expiry).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	}
	if len(l) == 0 {
		return nil
	}
	return s.firestoreClient.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		for i := range l {
			if err := tx.Delete(l[i].Ref, firestore.LastUpdateTime(l[i].UpdateTime)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) Watch(ctx context.Context) <-chan Event {
	ch := make(chan Event)
	go func() {
		defer close(ch)

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		if err := s.Init(ctx); err != nil {
			ch <- Event{
				Err: err,
			}
			return
		}

		itr := s.firestoreClient.Collection("clipboard").Query.
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
				ch <- Event{
					Err: err,
				}
				return
			}
			for _, dc := range qs.Changes {
				var evt Event
				switch dc.Kind {
				case firestore.DocumentAdded:
					evt.Type = EventCopy
				case firestore.DocumentModified:
					evt.Type = EventCopy
				case firestore.DocumentRemoved:
					evt.Type = EventRemove
				}
				var v Container
				if err := dc.Doc.DataTo(&v); err != nil {
					ch <- Event{
						Err: err,
					}
					return
				}
				evt.Value = v.Value
				ch <- evt
			}
		}
	}()
	return ch
}

func md5sum(s string) string {
	byt := md5.Sum([]byte(s))
	return hex.EncodeToString(byt[:])
}
