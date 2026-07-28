package callablenil

type Transform func(int32) int32
type Other func(int32) int32
type TransformAlias = Transform
type RawAlias = func(int32) int32

var Package Transform
var Trace int32

func Increment(value int32) int32 {
	return value + 1
}

func IsNilRaw(value func(int32) int32) bool {
	return value == nil
}

func IsNilDefined(value Transform) bool {
	return value == nil
}

func IsNonNilAlias(value TransformAlias) bool {
	return value != nil
}

func PackageIsNil() bool {
	return Package == nil
}

func LocalNil() bool {
	var callback Transform
	return callback == nil
}

func NilResult() Transform {
	return nil
}

func SetPackage(value Transform) {
	Package = value
}

func CallPackage(value int32) int32 {
	return Package(value)
}

func DefinedFromRaw(value func(int32) int32) Transform {
	return Transform(value)
}

func OtherFromDefined(value Transform) Other {
	return Other(value)
}

func TransformFromOther(value Other) Transform {
	return Transform(value)
}

func ConvertNilRaw() Transform {
	var value func(int32) int32
	return Transform(value)
}

func ConvertNilDefined() func(int32) int32 {
	var value Transform
	return (func(int32) int32)(value)
}

func ApplyDefined(value Transform, input int32) int32 {
	return value(input)
}

func ApplyRawAlias(value RawAlias, input int32) int32 {
	return value(input)
}

func PassRawToDefined(input int32) int32 {
	return ApplyDefined(Increment, input)
}

func ReturnRawAsDefined() Transform {
	return Increment
}

func ReturnDefinedAsRaw() func(int32) int32 {
	return DefinedFromRaw(Increment)
}

func AssignDefinedToRaw(value Transform, input int32) int32 {
	raw := (func(int32) int32)(value)
	return raw(input)
}

func LocalVarCall(value Transform, input int32) int32 {
	var result int32 = value(input)
	return result
}

func LocalImplicitDefined(input int32) int32 {
	var value Transform = Increment
	return value(input)
}

func Offset(delta int32) Transform {
	return Transform(func(value int32) int32 {
		return value + delta
	})
}

func ImplicitOffset(delta int32) Transform {
	return func(value int32) int32 {
		return value + delta
	}
}

func LocalDefined(value int32) int32 {
	type Local func(int32) int32
	callback := Local(Increment)
	return callback(value)
}

func NewRawIsNil() bool {
	pointer := new(func(int32) int32)
	return *pointer == nil
}

func NewDefinedIsNil() bool {
	pointer := new(Transform)
	return *pointer == nil
}

func StoreThroughPointer(value int32) int32 {
	pointer := new(func(int32) int32)
	*pointer = Increment
	return (*pointer)(value)
}

func mark(step int32) int32 {
	Trace = Trace*10 + step
	return step
}

func chooseBool(value bool) func() bool {
	mark(4)
	return func() bool {
		mark(5)
		return value
	}
}

func ShortCircuit(flag bool) int32 {
	ResetTrace()
	if flag && chooseBool(true)() {
		mark(6)
	}
	return Trace
}

func Conditional(value bool) int32 {
	ResetTrace()
	if chooseBool(value)() {
		mark(7)
	} else {
		mark(8)
	}
	return Trace
}

func chooseNil() func(int32) int32 {
	mark(1)
	return nil
}

func ResetTrace() {
	Trace = 0
}

func NilCallOrder() int32 {
	return chooseNil()(mark(2))
}

func NilVoidCallOrder() {
	var callback func(int32)
	callback(mark(3))
}

func TraceValue() int32 {
	return Trace
}
