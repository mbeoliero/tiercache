package rediscache

import jsoniter "github.com/json-iterator/go"

type Codec[T any] interface {
	Marshal(T) ([]byte, error)
	Unmarshal([]byte, *T) error
}

type JsonCodec[T any] struct{}

func (c *JsonCodec[T]) Marshal(v T) ([]byte, error) {
	return jsoniter.Marshal(v)
}

func (c *JsonCodec[T]) Unmarshal(data []byte, v *T) error {
	return jsoniter.Unmarshal(data, v)
}
