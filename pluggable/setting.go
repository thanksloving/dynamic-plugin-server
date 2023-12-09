package pluggable

import (
	"time"

	"github.com/patrickmn/go-cache"
)

var (
	defaultTimeout           = 1000 * time.Millisecond
	defaultCodec   Codec     = &MsgpackCodec{}
	defaultCache   Cacheable = &memoryCache{
		c: cache.New(5*time.Minute, 10*time.Minute),
	}
)

func SetCache(cache Cacheable) {
	defaultCache = cache
}

func SetDefaultTimeout(duration time.Duration) {
	defaultTimeout = duration
}

func SetDefaultCodec(codec Codec) {
	defaultCodec = codec
}
