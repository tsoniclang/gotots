package genericcallback

import "example.com/genericcallback/dependency"

func Apply[T any](value T, predicate func(T) bool) bool {
	return predicate(value)
}

func ApplyFirst[T any](value T, predicates []func(T) bool) bool {
	return predicates[0](value)
}

func Resolve[T any](callback func() T) T {
	value := callback()
	return value
}

type Box[T any] struct {
	Value T
}

type Sequence[T any] func(func(T) bool)

var InitializerApply = Apply("initializer", func(value string) bool {
	return value == "initializer"
})

var ChannelInitializerApply = Apply(int32(2), func(value int32) bool {
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

func FilterSequence[T any](
	values []T,
	predicate func(T) bool,
) Sequence[T] {
	return func(yield func(T) bool) {
		for _, value := range values {
			if predicate(value) && !yield(value) {
				return
			}
		}
	}
}

func PlainSequence() string {
	sequence := FilterSequence(
		[]string{"skip", "kept"},
		func(value string) bool { return value == "kept" },
	)
	var result string
	sequence(func(value string) bool {
		result += value
		return true
	})
	return result
}

func ChannelSequence() int32 {
	values := make(chan int32, 1)
	values <- 2
	sequence := FilterSequence(
		[]int32{1, 2},
		func(value int32) bool {
			return value == 2 && value == <-values
		},
	)
	var result int32
	sequence(func(value int32) bool {
		result += value
		return true
	})
	return result
}

func ChannelBoolSequence() bool {
	values := make(chan bool, 1)
	values <- true
	sequence := FilterSequence(
		[]bool{false, true},
		func(value bool) bool {
			return value && value == <-values
		},
	)
	var result bool
	sequence(func(value bool) bool {
		result = value
		return true
	})
	return result
}

func ChannelApply() bool {
	values := make(chan int32, 1)
	values <- 40
	return Apply(int32(2), func(value int32) bool {
		return value+<-values == 42
	})
}

func ChannelLexicalResult() int32 {
	type result struct {
		value int32
	}
	values := make(chan int32, 1)
	values <- 42
	resolved := Resolve(func() result {
		return result{value: <-values}
	})
	return resolved.value
}

func PlainApply() bool {
	return Apply("value", func(value string) bool {
		return value == "value"
	})
}

func ChannelFunctionValue() bool {
	values := make(chan int32, 1)
	values <- 40
	apply := Apply[int32]
	return apply(int32(2), func(value int32) bool {
		return value+<-values == 42
	})
}

func PlainFunctionValue() bool {
	apply := Apply[string]
	return apply("value", func(value string) bool {
		return value == "value"
	})
}

func ChannelNested() bool {
	values := make(chan int32, 1)
	values <- 40
	return ApplyFirst(int32(2), []func(int32) bool{
		func(value int32) bool {
			return value+<-values == 42
		},
	})
}

func ChannelMethod() bool {
	values := make(chan int32, 1)
	values <- 40
	return (Box[int32]{Value: 2}).Apply(func(value int32) bool {
		return value+<-values == 42
	})
}

func ChannelMethodValue() bool {
	values := make(chan int32, 1)
	values <- 40
	apply := (Box[int32]{Value: 2}).Apply
	return apply(func(value int32) bool {
		return value+<-values == 42
	})
}

func PlainMethodValue() bool {
	apply := (Box[string]{Value: "value"}).Apply
	return apply(func(value string) bool {
		return value == "value"
	})
}

func ChannelMethodExpression() bool {
	values := make(chan int32, 1)
	values <- 40
	apply := Box[int32].Apply
	return apply(Box[int32]{Value: 2}, func(value int32) bool {
		return value+<-values == 42
	})
}

func ChannelResult() int32 {
	values := make(chan int32, 1)
	values <- 9
	receive := MakeReceiver(values)
	return receive()
}

func Independent(value int64, predicate func(int64) bool) bool {
	return predicate(value)
}

func IndependentPlain() bool {
	return Independent(7, func(value int64) bool {
		return value == 7
	})
}

func InitializedPlainCallback() bool {
	return InitializerApply
}

func InitializedChannelCallback() bool {
	return ChannelInitializerApply
}

func IndependentPackageInitializer() int32 {
	return dependency.Value
}

func ChannelPredicateProvider(
	values <-chan int32,
) func(int32) bool {
	return func(value int32) bool {
		return value == <-values
	}
}

func IsSeven(value int32) bool {
	return value == 7
}

func InvokeIntPredicate(predicate func(int32) bool) bool {
	return predicate(7)
}

func NamedPlainApply() bool {
	return InvokeIntPredicate(IsSeven)
}

func GenericProfileWithNamedCallback[T any](
	value T,
	predicate func(T) bool,
) bool {
	return InvokeIntPredicate(IsSeven) && predicate(value)
}

func ChannelGenericProfileWithNamedCallback() bool {
	values := make(chan int32, 1)
	values <- 7
	return GenericProfileWithNamedCallback(
		int32(7),
		func(value int32) bool {
			return value == <-values
		},
	)
}

type CallbackMap[K any, V any] struct {
	Key   K
	Value V
}

func (m CallbackMap[K, V]) Range(callback func(K, V) bool) bool {
	return callback(m.Key, m.Value)
}

type CallbackSet[T any] struct {
	Values CallbackMap[T, struct{}]
}

func (s CallbackSet[T]) Range(callback func(T) bool) bool {
	return s.Values.Range(func(key T, _ struct{}) bool {
		return callback(key)
	})
}

func ChannelNestedGenericMethod() bool {
	values := make(chan int32, 1)
	values <- 7
	set := CallbackSet[int32]{
		Values: CallbackMap[int32, struct{}]{Key: 7},
	}
	return set.Range(func(value int32) bool {
		return value == <-values
	})
}

type RecursiveBox[T any] struct {
	Value T
	Proxy *RecursiveBox[T]
}

func (box *RecursiveBox[T]) Apply(callback func(T) bool) bool {
	if box.Proxy != nil {
		return box.Proxy.Apply(callback)
	}
	return callback(box.Value)
}

func ChannelRecursiveGenericMethod() bool {
	values := make(chan int32, 1)
	values <- 7
	leaf := &RecursiveBox[int32]{Value: 7}
	root := &RecursiveBox[int32]{Proxy: leaf}
	return root.Apply(func(value int32) bool {
		return value == <-values
	})
}

type CallbackHolder[T any] struct {
	Apply func(T) T
}

func NewCallbackHolder[T any](apply func(T) T) *CallbackHolder[T] {
	return &CallbackHolder[T]{Apply: apply}
}

func (holder *CallbackHolder[T]) Run(value T) T {
	return holder.Apply(value)
}

func CloneCallbackHolder[T any](
	holder *CallbackHolder[T],
) *CallbackHolder[T] {
	return &CallbackHolder[T]{Apply: holder.Apply}
}

func ChannelStoredCallback() int32 {
	values := make(chan int32, 1)
	values <- 40
	holder := NewCallbackHolder(func(value int32) int32 {
		return value + <-values
	})
	return holder.Run(2)
}

func PlainStoredCallback() string {
	holder := NewCallbackHolder(func(value string) string {
		return value + "!"
	})
	return holder.Run("value")
}

func ClonePlainStoredCallback() string {
	holder := NewCallbackHolder(func(value string) string {
		return value + "?"
	})
	return CloneCallbackHolder(holder).Run("clone")
}

type MutableValue[T any] interface {
	Change(func(T))
}

type MutableBox[T any] struct {
	Value T
}

func (box *MutableBox[T]) Change(apply func(T)) {
	apply(box.Value)
}

func ChannelGenericInterfaceMethod() int32 {
	values := make(chan int32, 1)
	values <- 40
	var target MutableValue[int32] = &MutableBox[int32]{Value: 2}
	var result int32
	target.Change(func(value int32) {
		result = value + <-values
	})
	return result
}

func PlainGenericInterfaceMethod() string {
	var target MutableValue[string] = &MutableBox[string]{Value: "value"}
	var result string
	target.Change(func(value string) {
		result = value
	})
	return result
}
