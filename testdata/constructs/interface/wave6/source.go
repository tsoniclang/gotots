package wave6interfaces

type Reader interface {
	Read(delta int32) int32
}

type Namer interface {
	Name() int32
}

type ReaderNamer interface {
	Reader
	Namer
}

type privateReader interface {
	hidden() int32
}

type Counter struct {
	Value int32
}

func (counter Counter) Read(delta int32) int32 {
	return counter.Value + delta
}

func (counter *Counter) Name() int32 {
	if counter == nil {
		return -1
	}
	return counter.Value
}

func (counter Counter) hidden() int32 {
	return counter.Value + 100
}

type Other struct {
	Value int32
}

type FloatKey float64

func (other Other) Read(delta int32) int32 {
	return other.Value + delta
}

type Wrapped struct {
	Counter
}

type InterfaceHolder struct {
	Value Reader
}

func classify(value any) int32 {
	switch selected := value.(type) {
	case nil:
		return 0
	case Counter:
		return selected.Value
	case ReaderNamer:
		return selected.Read(0) + selected.Name() + 100
	case Reader:
		return selected.Read(0) + 200
	case int32, string:
		if selected == nil {
			return -2
		}
		return 300
	default:
		return -1
	}
}

func interfaceCalls() []int32 {
	counter := Counter{Value: 4}
	var reader Reader = counter
	counter.Value = 20
	method := reader.Read
	expression := Reader.Read

	pointer := &Counter{Value: 7}
	var combined ReaderNamer = pointer
	pointer.Value = 8
	var subset Reader = combined

	var nilPointer *Counter
	var typedNil Namer = nilPointer
	var nilInterface Namer

	return []int32{
		reader.Read(1),
		method(2),
		expression(reader, 3),
		combined.Read(1),
		combined.Name(),
		subset.Read(2),
		boolValue(typedNil == nil),
		typedNil.Name(),
		boolValue(nilInterface == nil),
	}
}

func assertions() []int32 {
	var value any = Counter{Value: 9}
	concrete, concreteOK := value.(Counter)
	reader, readerOK := value.(Reader)
	namer, namerOK := value.(Namer)
	anonymous, anonymousOK := value.(interface {
		Read(int32) int32
	})

	var pointer any = &Counter{Value: 11}
	pointerValue, pointerOK := pointer.(*Counter)
	combined, combinedOK := pointer.(ReaderNamer)

	return []int32{
		concrete.Value,
		boolValue(concreteOK),
		reader.Read(1),
		boolValue(readerOK),
		boolValue(namerOK),
		boolValue(namer == nil),
		anonymous.Read(2),
		boolValue(anonymousOK),
		pointerValue.Value,
		boolValue(pointerOK),
		combined.Name(),
		boolValue(combinedOK),
	}
}

func equalityAndMaps() []int32 {
	var first any = Counter{Value: 3}
	var same any = Counter{Value: 3}
	var differentValue any = Counter{Value: 4}
	var differentType any = Other{Value: 3}
	var nilValue any
	var nilPointer *Counter
	var typedNil any = nilPointer

	values := map[any]int32{
		Counter{Value: 3}: 10,
		int32(4):          20,
		"five":            30,
	}
	pointer := &Counter{Value: 8}
	pointerAlias := pointer
	otherPointer := &Counter{Value: 8}
	var nilPointerKey *Counter
	floatKey := float64(1.25)
	complexKey := complex128(2 + 3i)
	arrayKey := [2]int32{1, 2}
	values[pointer] = 40
	values[nilPointerKey] = 50
	values[floatKey] = 60
	values[complexKey] = 70
	values[arrayKey] = 80
	values[nil] = 90

	return []int32{
		boolValue(first == same),
		boolValue(first == differentValue),
		boolValue(first == differentType),
		boolValue(nilValue == nil),
		boolValue(typedNil == nil),
		values[Counter{Value: 3}],
		values[int32(4)],
		values["five"],
		values[pointerAlias],
		values[otherPointer],
		values[nilPointerKey],
		values[float64(1.25)],
		values[complex128(2+3i)],
		values[[2]int32{1, 2}],
		values[nil],
	}
}

func directFloatMaps() []int32 {
	values := map[float64]int32{1.25: 10}
	var zero float64
	negativeZero := -zero
	values[negativeZero] = 20
	notANumber := zero / zero
	values[notANumber] = 30
	_, foundNaN := values[notANumber]

	defined := map[FloatKey]int32{FloatKey(2.5): 40}
	return []int32{
		values[1.25],
		values[zero],
		boolValue(foundNaN),
		int32(len(values)),
		defined[FloatKey(2.5)],
	}
}

func FailedAssertion() {
	var value any = Counter{Value: 1}
	_ = value.(Namer)
}

func UncomparableEquality() {
	var left any = []int32{1}
	_ = left == left
}

func UnhashableMapKey() {
	values := make(map[any]int32)
	var key any = []int32{1}
	values[key] = 1
}

func typeSwitches() []int32 {
	var pointer any = &Counter{Value: 6}
	var reader any = Other{Value: 7}
	return []int32{
		classify(nil),
		classify(Counter{Value: 5}),
		classify(pointer),
		classify(reader),
		classify(int32(8)),
		classify(true),
	}
}

func promotedAdapters() []int32 {
	var value Reader = Wrapped{Counter: Counter{Value: 12}}
	var private privateReader = Wrapped{Counter: Counter{Value: 13}}
	holder := InterfaceHolder{Value: value}
	return []int32{
		holder.Value.Read(1),
		private.hidden(),
	}
}

func localInterfaces() []int32 {
	type Local interface {
		Read(int32) int32
	}
	type LocalValue int32
	type LocalRecord struct {
		Value int32
	}

	var selected Local = Counter{Value: 14}
	var broad any = LocalValue(16)
	concrete, concreteOK := broad.(LocalValue)
	var record any = LocalRecord{Value: 17}
	recordValue, recordOK := record.(LocalRecord)
	records := map[any]int32{LocalRecord{Value: 17}: 18}
	_, impossible := broad.(interface {
		Take(LocalValue)
	})
	return []int32{
		selected.Read(1),
		int32(concrete),
		boolValue(concreteOK),
		recordValue.Value,
		boolValue(recordOK),
		records[LocalRecord{Value: 17}],
		boolValue(impossible),
	}
}

func makeLocalRecord(value int32) any {
	type LocalRecord struct {
		Value int32
	}
	return LocalRecord{Value: value}
}

func localDynamicIdentity() []int32 {
	first := makeLocalRecord(21)
	same := makeLocalRecord(21)
	different := makeLocalRecord(22)
	values := map[any]int32{first: 23}
	return []int32{
		boolValue(first == same),
		boolValue(first == different),
		values[same],
	}
}

func boolValue(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

func Audit() []int32 {
	result := interfaceCalls()
	result = append(result, assertions()...)
	result = append(result, equalityAndMaps()...)
	result = append(result, directFloatMaps()...)
	result = append(result, typeSwitches()...)
	result = append(result, promotedAdapters()...)
	result = append(result, localInterfaces()...)
	result = append(result, localDynamicIdentity()...)
	return result
}
