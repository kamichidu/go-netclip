package clipboard

import (
	"crypto/md5"
	"encoding/hex"
)

// func (v *Container) String() string {
// 	ts := v.Timestamp.Format(time.RFC3339)
// 	return ts + " - " + Shorten(v.Value)
// }

func MD5(s string) string {
	byt := md5.Sum([]byte(s))
	return hex.EncodeToString(byt[:])
}
