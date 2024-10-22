package clipboard

import (
	"time"
)

type Container struct {
	Value string `firestore:"value"`

	MD5 string `firestore:"md5"`

	Timestamp time.Time `firestore:"timestamp"`
}

func (v *Container) String() string {
	ts := v.Timestamp.Format(time.RFC3339)
	return ts + " - " + Shorten(v.Value)
}
