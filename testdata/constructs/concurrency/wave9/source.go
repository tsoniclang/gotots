package wave9concurrency

func Buffered() int32 {
	values := make(chan int32, 1)
	values <- 7
	return <-values
}

func Unbuffered() int32 {
	values := make(chan int32)
	go func() {
		values <- 9
	}()
	return <-values
}

func Ordering() int32 {
	buffered := make(chan int32, 2)
	buffered <- 1
	buffered <- 2
	firstBuffered := <-buffered
	secondBuffered := <-buffered

	unbuffered := make(chan int32)
	go func() {
		unbuffered <- 3
		unbuffered <- 4
	}()
	firstUnbuffered := <-unbuffered
	secondUnbuffered := <-unbuffered
	return firstBuffered*1000 +
		secondBuffered*100 +
		firstUnbuffered*10 +
		secondUnbuffered
}

func CloseDrain() int32 {
	values := make(chan int32, 1)
	values <- 5
	close(values)
	first, firstOK := <-values
	second, secondOK := <-values
	result := first + second
	if firstOK {
		result += 10
	}
	if secondOK {
		result += 100
	}
	return result
}

func DirectionAndMeasure() int32 {
	values := make(chan int32, 2)
	var send chan<- int32 = values
	var receive <-chan int32 = values
	send <- 6
	result := int32(len(receive)*10 + cap(send))
	if receive == (<-chan int32)(values) {
		result += <-receive
	}
	return result
}

func ChannelRange() int32 {
	values := make(chan int32, 2)
	values <- 1
	values <- 2
	close(values)
	var result int32
	for value := range values {
		result += value
	}
	return result
}

func SelectDefault() int32 {
	var values <-chan int32
	select {
	case <-values:
		return 0
	default:
		return 11
	}
}

func SelectReceive() int32 {
	values := make(chan int32, 1)
	values <- 12
	select {
	case value, ok := <-values:
		if ok {
			return value
		}
	}
	return 0
}

func SelectSend() int32 {
	values := make(chan int32, 1)
	select {
	case values <- 13:
	default:
		return 0
	}
	return <-values
}

func SelectRendezvous() int32 {
	values := make(chan int32)
	done := make(chan int32, 1)
	go func() {
		select {
		case values <- 17:
			done <- 1
		}
	}()
	var result int32
	select {
	case result = <-values:
	}
	<-done
	return result
}

func bump(value *int32) int32 {
	*value++
	return 0
}

func SelectDelayedTarget() int32 {
	var blocked <-chan int32
	var calls int32
	result := []int32{0}
	select {
	case result[bump(&calls)] = <-blocked:
	default:
	}
	ready := make(chan int32, 1)
	ready <- 8
	select {
	case result[bump(&calls)] = <-ready:
	}
	return calls*10 + result[0]
}

func prepared(value int32) <-chan int32 {
	values := make(chan int32, 1)
	values <- value
	return values
}

func receiveOne(values <-chan int32) int32 {
	return <-values
}

func synchronousOne(values <-chan int32) int32 {
	return 20
}

func invoke(
	selected func(<-chan int32) int32,
	values <-chan int32,
) int32 {
	return selected(values)
}

func returned() func(<-chan int32) int32 {
	return receiveOne
}

type FunctionHolder struct {
	Run func(<-chan int32) int32
}

type Receiver struct {
	Values <-chan int32
}

func (receiver *Receiver) Next() int32 {
	return <-receiver.Values
}

type Reader interface {
	Next() int32
}

type ImmediateReceiver struct{}

func (*ImmediateReceiver) Next() int32 {
	return 31
}

func DirectSynchronousInterface() int32 {
	var reader Reader = &ImmediateReceiver{}
	return reader.Next()
}

type GenericValue[T any] interface {
	Value() T
}

type IntValue interface {
	Value() int32
}

type BlockingIntValue struct {
	Values <-chan int32
}

func (value *BlockingIntValue) Value() int32 {
	return <-value.Values
}

type ImmediateStringValue struct{}

func (*ImmediateStringValue) Value() string {
	return "ok"
}

func readGenericValue[T any](value GenericValue[T]) T {
	return value.Value()
}

func readIntValue(value IntValue) int32 {
	return value.Value()
}

func GenericInterfaceAudit() int32 {
	values := make(chan int32, 2)
	values <- 37
	values <- 5
	number := readGenericValue[int32](
		&BlockingIntValue{Values: values},
	)
	number += readIntValue(&BlockingIntValue{Values: values})
	text := readGenericValue[string](&ImmediateStringValue{})
	if text != "ok" {
		return -1
	}
	return number
}

func identity[T any](value T) T {
	return value
}

func Transport() int32 {
	local := receiveOne
	result := local(prepared(1))
	result += invoke(receiveOne, prepared(2))
	result += returned()(prepared(3))

	holder := FunctionHolder{Run: receiveOne}
	result += holder.Run(prepared(4))
	pointer := &holder
	result += pointer.Run(prepared(5))

	array := [1]func(<-chan int32) int32{receiveOne}
	result += array[0](prepared(6))
	slice := []func(<-chan int32) int32{receiveOne}
	result += slice[0](prepared(7))
	mapping := map[int32]func(<-chan int32) int32{0: receiveOne}
	result += mapping[0](prepared(8))

	var boxed any = receiveOne
	asserted := boxed.(func(<-chan int32) int32)
	result += asserted(prepared(9))
	generic := identity[func(<-chan int32) int32](receiveOne)
	result += generic(prepared(10))
	result += PackageReceiver(prepared(14))
	nested := identity[[]func(<-chan int32) int32](
		[]func(<-chan int32) int32{receiveOne},
	)
	result += nested[0](prepared(15))

	receiver := &Receiver{Values: prepared(11)}
	methodValue := receiver.Next
	result += methodValue()
	methodExpression := (*Receiver).Next
	receiver = &Receiver{Values: prepared(12)}
	result += methodExpression(receiver)
	var reader Reader = &Receiver{Values: prepared(13)}
	result += reader.Next()

	synchronous := []func(<-chan int32) int32{
		receiveOne,
		synchronousOne,
	}
	result += synchronous[1](nil)
	return result
}

func closure(values <-chan int32) func() int32 {
	return func() int32 {
		return <-values
	}
}

func AggregateClosures() int32 {
	array := [1]func() int32{closure(prepared(1))}
	slice := []func() int32{closure(prepared(2))}
	mapping := map[int32]func() int32{0: closure(prepared(3))}
	holder := struct {
		Run func() int32
	}{Run: closure(prepared(4))}
	return array[0]() + slice[0]() + mapping[0]() + holder.Run()
}

func cycleA(values <-chan int32, depth int32) int32 {
	if depth == 0 {
		return <-values
	}
	return cycleB(values, depth-1)
}

func cycleB(values <-chan int32, depth int32) int32 {
	return cycleA(values, depth)
}

func Recursive() int32 {
	return cycleA(prepared(14), 2)
}

func appendBang(value string) string {
	return value + "!"
}

func WhollySynchronous() string {
	selected := appendBang
	return selected("ok")
}

func Audit() (
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int32,
	int16,
	int64,
	int32,
) {
	return Buffered(),
		Unbuffered(),
		Ordering(),
		CloseDrain(),
		DirectionAndMeasure(),
		ChannelRange(),
		SelectDefault(),
		SelectReceive(),
		SelectSend(),
		SelectRendezvous(),
		SelectDelayedTarget(),
		Transport(),
		AggregateClosures(),
		Recursive(),
		ValueRecursive(),
		DirectSynchronous(),
		DiscardedReceive(),
		ChannelIdentityAndCopy(),
		GenericChannel(),
		GoroutineEvaluation(),
		GoroutineForms(),
		SelectEvaluation(),
		SelectControl(),
		DeferCooperative(),
		GenericConstraintChannel(),
		GenericConstraintForward(),
		GenericConstraintSynchronous(),
		DeferRecoverCooperative()
}
