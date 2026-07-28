package consumer

import "example.com/definedcallable/producer"

type Alias = producer.Transform

func Use(value int32) int32 {
	return producer.Apply(producer.Make(), value)
}

func FromRaw(value int32) int32 {
	var transform producer.Transform = producer.Increment
	return producer.Apply(transform, value)
}

func IsNil(transform Alias) bool {
	return transform == nil
}
