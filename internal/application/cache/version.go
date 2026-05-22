package cache

import (
	"strconv"
	"time"
)

func newBlacklistVersion() string {
	return strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
}
