package wave9concurrency

type deferredAction interface {
	Apply()
}

type deferredReceiver struct {
	Result *int32
	Values <-chan int32
}

func (receiver *deferredReceiver) Apply() {
	*receiver.Result = *receiver.Result*10 + <-receiver.Values
}

func deferredFunction(result *int32, values <-chan int32) {
	*result = *result*10 + <-values
}

func deferredGeneric[T any](result *T, values <-chan T) {
	*result = <-values
}

func deferredOnly(
	direct <-chan int32,
	value <-chan int32,
	method <-chan int32,
	dynamic <-chan int32,
	generic <-chan int32,
) (result int32) {
	defer deferredFunction(&result, direct)
	selected := deferredFunction
	defer selected(&result, value)
	receiver := &deferredReceiver{Result: &result, Values: method}
	defer receiver.Apply()
	var action deferredAction = &deferredReceiver{
		Result: &result,
		Values: dynamic,
	}
	defer action.Apply()
	var genericResult int32
	defer func() {
		result = result*10 + genericResult
	}()
	defer deferredGeneric[int32](&genericResult, generic)
	return result
}

func DeferCooperative() int32 {
	return deferredOnly(
		prepared(1),
		prepared(2),
		prepared(3),
		prepared(4),
		prepared(5),
	)
}

func cooperativeDeferredRecover(values <-chan int32) int32 {
	defer recover()
	return <-values
}

func DeferRecoverCooperative() int32 {
	return cooperativeDeferredRecover(prepared(9))
}

type constraintPuller interface {
	Pull() int32
}

type channelPuller struct {
	Values <-chan int32
}

func (puller *channelPuller) Pull() int32 {
	return <-puller.Values
}

func pullConstraint[T constraintPuller](value T) int32 {
	return value.Pull()
}

func GenericConstraintChannel() int32 {
	return pullConstraint[*channelPuller](
		&channelPuller{Values: prepared(6)},
	)
}

type forwardPuller interface {
	ForwardPull() int16
}

type channelForwardPuller struct {
	Values <-chan int16
}

func (puller *channelForwardPuller) ForwardPull() int16 {
	return <-puller.Values
}

func forwardLeaf[T forwardPuller](value T) int16 {
	return value.ForwardPull()
}

func forwardBridge[T forwardPuller](value T) int16 {
	return forwardLeaf[T](value)
}

func GenericConstraintForward() int16 {
	values := make(chan int16, 1)
	values <- 7
	return forwardBridge[*channelForwardPuller](
		&channelForwardPuller{Values: values},
	)
}

type staticReader interface {
	ReadStatic() int64
}

type staticValue struct {
	Value int64
}

func (value *staticValue) ReadStatic() int64 {
	return value.Value
}

func readStatic[T staticReader](value T) int64 {
	return value.ReadStatic()
}

func GenericConstraintSynchronous() int64 {
	return readStatic[*staticValue](&staticValue{Value: 8})
}
