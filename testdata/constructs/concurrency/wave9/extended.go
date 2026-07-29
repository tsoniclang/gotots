package wave9concurrency

var PackageReceiver = receiveOne

func valueCycleA(
	values <-chan int32,
	selected func(<-chan int32) int32,
	depth int32,
) int32 {
	if depth == 0 {
		return selected(values)
	}
	return valueCycleB(values, selected, depth-1)
}

func valueCycleB(
	values <-chan int32,
	selected func(<-chan int32) int32,
	depth int32,
) int32 {
	return valueCycleA(values, selected, depth)
}

func ValueRecursive() int32 {
	return valueCycleA(prepared(16), receiveOne, 2)
}

func DirectSynchronous() int32 {
	return synchronousOne(nil)
}

func DiscardedReceive() int32 {
	values := make(chan int32, 1)
	values <- 1
	<-values
	return int32(len(values))
}

type NamedChannel chan int32

type Payload struct {
	Value int32
}

func ChannelIdentityAndCopy() int32 {
	values := make(chan Payload, 1)
	payload := Payload{Value: 3}
	values <- payload
	payload.Value = 9
	result := (<-values).Value

	named := make(NamedChannel, 1)
	named <- 4
	projected := (chan int32)(named)
	mapping := map[chan int32]int32{projected: 5}
	if projected == (chan int32)(named) {
		result += mapping[projected]
	}
	if projected == nil {
		result += 100
	}
	if projected == make(chan int32) {
		result += 1000
	}
	result += <-named

	closed := make(chan Payload)
	close(closed)
	zero, ok := <-closed
	if !ok {
		result += zero.Value + 10
	}
	return result
}

func genericReceive[T any](values <-chan T) T {
	return <-values
}

func GenericChannel() int32 {
	values := make(chan int32, 1)
	values <- 17
	return genericReceive[int32](values)
}

func markPayload(log *int32, value Payload) Payload {
	*log = *log*10 + 2
	return value
}

func selectSender(
	log *int32,
	values chan<- int32,
) func(Payload) {
	*log = *log*10 + 1
	return func(value Payload) {
		values <- value.Value
	}
}

func GoroutineEvaluation() int32 {
	values := make(chan int32, 1)
	var log int32
	payload := Payload{Value: 7}
	go selectSender(&log, values)(markPayload(&log, payload))
	payload.Value = 9
	snapshot := log
	return snapshot*10 + <-values
}

type Sender interface {
	Put(int32)
}

type ChannelSender struct {
	Values chan<- int32
}

func (sender *ChannelSender) Put(value int32) {
	sender.Values <- value
}

func genericSend[T any](values chan<- T, value T) {
	values <- value
}

func GoroutineForms() int32 {
	values := make(chan int32, 4)
	sender := &ChannelSender{Values: values}
	method := sender.Put
	var dynamic Sender = sender
	function := func(value int32) {
		values <- value
	}
	go method(1)
	go dynamic.Put(2)
	go function(3)
	go genericSend[int32](values, 4)
	var result int32
	for range 4 {
		result += <-values
	}
	return result
}

func selectChannel(
	log *int32,
	digit int32,
	values chan int32,
) chan int32 {
	*log = *log*10 + digit
	return values
}

func selectValue(log *int32, digit int32, value int32) int32 {
	*log = *log*10 + digit
	return value
}

func SelectEvaluation() int32 {
	left := make(chan int32, 1)
	right := make(chan int32)
	var log int32
	select {
	case selectChannel(&log, 1, left) <- selectValue(&log, 2, 3):
	case <-selectChannel(&log, 3, right):
	default:
	}
	return log*10 + <-left
}

func SelectControl() int32 {
	values := make(chan int32, 2)
	values <- 2
	values <- 3
	var result int32
loop:
	for {
		select {
		case value := <-values:
			result += value
			if result == 2 {
				continue loop
			}
			break loop
		}
	}
	return result
}
