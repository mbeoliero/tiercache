package rediscache

import (
	"github.com/mbeoliero/tiercache/cacher"
	"github.com/mbeoliero/tiercache/codec"
)

type Option[K comparable, V any] struct {
	Codec  codec.Codec[V]
	Logger Logger
	Mws    []cacher.Middleware[K, V]
}

func defaultOption[K comparable, V any]() *Option[K, V] {
	return &Option[K, V]{
		Codec: &codec.JsonCodec[V]{},
	}
}
