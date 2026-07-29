package wave8control

import "runtime"

var trace string

func ControlFree(value int) int {
	return value + 1
}

func record(value string) {
	trace += value
}

func DeferOrder() (string, int) {
	trace = ""
	value := 3
	defer record("first")
	defer func(captured int) {
		record("second")
		value += captured
	}(value)
	value = 7
	return trace, value
}

func RecoverPanic() (result string) {
	defer func() {
		if recover() != nil {
			result = "recovered"
		}
	}()
	panic("boom")
}

func recoverOneCallBelow() any {
	return recover()
}

func RecoverDirectness() (direct bool, indirect bool) {
	defer func() {
		direct = recover() != nil
	}()
	defer func() {
		indirect = recoverOneCallBelow() != nil
		panic("direct")
	}()
	panic("initial")
}

func PanicNilIdentity() (matched bool) {
	defer func() {
		_, matched = recover().(*runtime.PanicNilError)
	}()
	panic(nil)
}

type recoveredError interface {
	Error() string
}

func PanicNilContracts() (
	asError bool,
	asErrorContract bool,
	asRuntime bool,
	asPanicNil bool,
	message string,
) {
	defer func() {
		value := recover()
		selectedError, ok := value.(error)
		asError = ok
		if ok {
			message = selectedError.Error()
		}
		selectedContract, ok := value.(recoveredError)
		asErrorContract = ok
		if ok {
			message = selectedContract.Error()
		}
		selectedRuntime, ok := value.(runtime.Error)
		asRuntime = ok
		if ok {
			selectedRuntime.RuntimeError()
		}
		_, asPanicNil = value.(*runtime.PanicNilError)
	}()
	panic(nil)
}

func RuntimeFaultIdentity() (
	asError bool,
	asErrorContract bool,
	asRuntime bool,
	asPanicNil bool,
	message string,
) {
	defer func() {
		value := recover()
		selected, ok := value.(error)
		asError = ok
		if ok {
			message = selected.Error()
		}
		selectedContract, ok := value.(recoveredError)
		asErrorContract = ok
		if ok {
			message = selectedContract.Error()
		}
		selectedRuntime, ok := value.(runtime.Error)
		asRuntime = ok
		if ok {
			selectedRuntime.RuntimeError()
		}
		_, asPanicNil = value.(*runtime.PanicNilError)
	}()
	values := []int{1}
	_ = values[2]
	return
}

type deferBox struct {
	Value int
}

func chooseDeferred() func(deferBox, string) {
	record("c")
	return func(value deferBox, argument string) {
		record(argument)
		if value.Value == 1 {
			record("1")
		} else {
			record("2")
		}
	}
}

func deferredArgument() string {
	record("a")
	return "x"
}

func DeferEvaluationAndCopy() (result string) {
	trace = ""
	value := deferBox{Value: 1}
	defer func() {
		result = trace
	}()
	defer chooseDeferred()(value, deferredArgument())
	value.Value = 2
	record("b")
	return
}

func selectedReceiver(value *deferBox) deferBox {
	record("r")
	return *value
}

func (value deferBox) observe() {
	if value.Value == 1 {
		record("1")
	} else {
		record("2")
	}
}

func (value *deferBox) observePointer() {
	if value.Value == 1 {
		record("1")
	} else {
		record("2")
	}
}

func DeferReceiverCopy() (result string) {
	trace = ""
	value := deferBox{Value: 1}
	defer func() {
		result = trace
	}()
	defer selectedReceiver(&value).observe()
	value.Value = 2
	record("b")
	return
}

func DeferPointerReceiver() (result string) {
	trace = ""
	value := deferBox{Value: 1}
	defer func() {
		result = trace
	}()
	defer value.observePointer()
	value.Value = 2
	record("b")
	return
}

func variadicDeferred(prefix string, values ...deferBox) {
	record(prefix)
	for _, value := range values {
		if value.Value == 1 {
			record("1")
		} else {
			record("2")
		}
	}
}

func multipleDeferred() (deferBox, string) {
	record("p")
	return deferBox{Value: 1}, "m"
}

func receiveMultipleDeferred(value deferBox, mark string) {
	record(mark)
	if value.Value == 1 {
		record("1")
	} else {
		record("2")
	}
}

func DeferVariadicAndMultiple() (result string) {
	trace = ""
	value := deferBox{Value: 1}
	values := []deferBox{value}
	defer func() {
		result = trace
	}()
	defer receiveMultipleDeferred(multipleDeferred())
	defer variadicDeferred("v", value)
	defer variadicDeferred("s", values...)
	value.Value = 2
	values[0].Value = 2
	record("b")
	return
}

func DeferBuiltins() (mapCleared bool, copied int, sliceCleared bool) {
	values := map[string]int{"a": 1, "b": 2}
	selectedMap := values
	key := "a"
	target := []int{0, 0}
	selectedTarget := target
	source := []int{1, 2}
	selectedSource := source
	cleared := []int{3, 4}
	selectedCleared := cleared
	defer func() {
		mapCleared = len(values) == 0
		copied = target[0]*10 + target[1]
		sliceCleared = cleared[0] == 0 && cleared[1] == 0
	}()
	defer clear(selectedCleared)
	defer copy(selectedTarget, selectedSource)
	defer clear(selectedMap)
	defer delete(selectedMap, key)
	selectedMap = nil
	key = "b"
	selectedTarget = nil
	selectedSource = nil
	selectedCleared = nil
	source[0] = 7
	return
}

func recoverFunction(mark string) {
	if recover() != nil {
		record(mark)
	}
}

type recoveryValue struct {
	Mark string
}

func (value recoveryValue) catch() {
	if recover() != nil {
		record(value.Mark)
	}
}

type recoveryContract interface {
	catch()
}

func genericRecover[T any](mark string) {
	if recover() != nil {
		record(mark)
	}
}

func recoveredTrace(invoke func()) (result string) {
	trace = ""
	defer func() {
		result = trace
	}()
	invoke()
	return
}

func recoverDirectFunction() {
	defer recoverFunction("d")
	panic("direct")
}

func recoverFunctionValue() {
	selected := recoverFunction
	defer selected("f")
	panic("function-value")
}

func recoverDirectMethod() {
	value := recoveryValue{Mark: "m"}
	defer value.catch()
	panic("method")
}

func recoverMethodValue() {
	value := recoveryValue{Mark: "v"}
	selected := value.catch
	defer selected()
	panic("method-value")
}

func recoverMethodExpression() {
	selected := recoveryValue.catch
	defer selected(recoveryValue{Mark: "e"})
	panic("method-expression")
}

func recoverInterfaceMethod() {
	var selected recoveryContract = recoveryValue{Mark: "i"}
	defer selected.catch()
	panic("interface")
}

func recoverGenericFunction() {
	defer genericRecover[string]("g")
	panic("generic")
}

type recoveryDefined func(string)

func recoverDefinedFunction() {
	selected := recoveryDefined(recoverFunction)
	defer selected("t")
	panic("defined")
}

type recoveryHolder struct {
	Call func(string)
}

func recoverFunctionField() {
	holder := recoveryHolder{Call: recoverFunction}
	defer holder.Call("h")
	panic("field")
}

func recoverFunctionPointer() {
	selected := recoverFunction
	pointer := &selected
	defer (*pointer)("p")
	panic("pointer")
}

func recoverFunctionSlice() {
	selected := []func(string){recoverFunction}
	defer selected[0]("s")
	panic("slice")
}

func RecoveryCallableForms() string {
	return recoveredTrace(recoverDirectFunction) +
		recoveredTrace(recoverFunctionValue) +
		recoveredTrace(recoverDirectMethod) +
		recoveredTrace(recoverMethodValue) +
		recoveredTrace(recoverMethodExpression) +
		recoveredTrace(recoverInterfaceMethod) +
		recoveredTrace(recoverGenericFunction) +
		recoveredTrace(recoverDefinedFunction) +
		recoveredTrace(recoverFunctionField) +
		recoveredTrace(recoverFunctionPointer) +
		recoveredTrace(recoverFunctionSlice)
}

func RecoverOnNormalReturn() (nilValue bool) {
	defer func() {
		nilValue = recover() == nil
	}()
	return
}

func RecoverOutsideDefer() bool {
	return recover() == nil
}

func RecoverContinuesUnwind() (result string) {
	trace = ""
	defer func() {
		result = trace
	}()
	defer record("o")
	defer func() {
		if recover() != nil {
			record("r")
		}
	}()
	panic("continue")
}

func ReplacementPanic() (result string) {
	defer func() {
		value := recover()
		if value == "replacement" {
			result = "replacement"
		}
	}()
	defer func() {
		panic("replacement")
	}()
	panic("initial")
}

func NamedResultMutation() (value int) {
	defer func() {
		value += 2
	}()
	return 5
}

func OrdinaryReturnUnwind() (result string) {
	trace = ""
	defer func() {
		result = trace
	}()
	defer record("d")
	record("b")
	return
}

func NilDeferredTiming() (reached bool, recovered bool) {
	defer func() {
		recovered = recover() != nil
	}()
	var selected func()
	defer selected()
	reached = true
	return
}

func NilValueReceiverTiming() (reached bool, recovered bool) {
	defer func() {
		recovered = recover() != nil
	}()
	var selected *deferBox
	defer selected.observe()
	reached = true
	return
}

func NilInterfaceDeferredTiming() (reached bool, recovered bool) {
	defer func() {
		recovered = recover() != nil
	}()
	var selected recoveryContract
	defer selected.catch()
	reached = true
	return
}
