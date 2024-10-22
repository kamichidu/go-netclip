package clipboard

import (
	"context"
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

type Container struct {
	Value string `firestore:"value"`

	Timestamp time.Time `firestore:"timestamp"`
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

func (s *Store) Copy(ctx context.Context, v string) error {
	if err := s.Init(ctx); err != nil {
		return err
	}

	value := &Container{
		Value:     v,
		Timestamp: time.Now(),
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
