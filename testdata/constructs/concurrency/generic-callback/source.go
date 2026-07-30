package genericcallback

import "example.com/genericcallback/dependency"

func Apply[T any](value T, predicate func(T) bool) bool {
	return predicate(value)
}

func ApplyFirst[T any](value T, predicates []func(T) bool) bool {
	return predicates[0](value)
}

type Box[T any] struct {
	Value T
}

var InitializerApply = Apply("initializer", func(value string) bool {
	return value == "initializer"
})

var CooperativeInitializerApply = Apply(int32(2), func(value int32) bool {
	return value+<-dependency.Values == 42
})

func (box Box[T]) Apply(predicate func(T) bool) bool {
	return predicate(box.Value)
}

func MakeReceiver[T any](values <-chan T) func() T {
	return func() T {
		return <-values
	}
}

func CooperativeApply() bool {
	values := make(chan int32, 1)
	values <- 40
	return Apply(int32(2), func(value int32) bool {
		return value+<-values == 42
	})
}

func SynchronousApply() bool {
	return Apply("value", func(value string) bool {
		return value == "value"
	})
}

func CooperativeFunctionValue() bool {
	values := make(chan int32, 1)
	values <- 40
	apply := Apply[int32]
	return apply(int32(2), func(value int32) bool {
		return value+<-values == 42
	})
}

func SynchronousFunctionValue() bool {
	apply := Apply[string]
	return apply("value", func(value string) bool {
		return value == "value"
	})
}

func CooperativeNested() bool {
	values := make(chan int32, 1)
	values <- 40
	return ApplyFirst(int32(2), []func(int32) bool{
		func(value int32) bool {
			return value+<-values == 42
		},
	})
}

func CooperativeMethod() bool {
	values := make(chan int32, 1)
	values <- 40
	return (Box[int32]{Value: 2}).Apply(func(value int32) bool {
		return value+<-values == 42
	})
}

func CooperativeMethodValue() bool {
	values := make(chan int32, 1)
	values <- 40
	apply := (Box[int32]{Value: 2}).Apply
	return apply(func(value int32) bool {
		return value+<-values == 42
	})
}

func SynchronousMethodValue() bool {
	apply := (Box[string]{Value: "value"}).Apply
	return apply(func(value string) bool {
		return value == "value"
	})
}

func CooperativeMethodExpression() bool {
	values := make(chan int32, 1)
	values <- 40
	apply := Box[int32].Apply
	return apply(Box[int32]{Value: 2}, func(value int32) bool {
		return value+<-values == 42
	})
}

func CooperativeResult() int32 {
	values := make(chan int32, 1)
	values <- 9
	receive := MakeReceiver(values)
	return receive()
}

func Independent(value int64, predicate func(int64) bool) bool {
	return predicate(value)
}

func IndependentSynchronous() bool {
	return Independent(7, func(value int64) bool {
		return value == 7
	})
}

func InitializedSynchronousCallback() bool {
	return InitializerApply
}

func InitializedCooperativeCallback() bool {
	return CooperativeInitializerApply
}

func IndependentPackageInitializer() int32 {
	return dependency.Value
}
