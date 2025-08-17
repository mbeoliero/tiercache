package rediscache

type Option[K comparable, T any] struct {
	Codec  Codec[T]
	Logger Logger
}

func defaultOption[K comparable, T any]() *Option[K, T] {
	return &Option[K, T]{
		Codec: &JsonCodec[T]{},
	}
}
