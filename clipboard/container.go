package clipboard

import (
	"crypto/md5"
	"encoding/hex"
	"time"
)

type Container struct {
	Value string

	Timestamp time.Time
}

func (v *Container) String() string {
	ts := v.Timestamp.Format(time.RFC3339)
	return ts + " - " + Shorten(v.Value)
}

func (v *Container) MD5() string {
	byt := md5.Sum([]byte(v.Value))
	return hex.EncodeToString(byt[:])
}
