package rediscache

import "github.com/mbeoliero/tiercache/store"

type Option[K comparable, V any] struct {
	Codec  Codec[V]
	Logger Logger
	Mws    []store.Middleware[K, V]
}

func defaultOption[K comparable, V any]() *Option[K, V] {
	return &Option[K, V]{
		Codec: &JsonCodec[V]{},
	}
}
