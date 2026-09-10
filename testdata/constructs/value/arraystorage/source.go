package arraystorage

type Pair [2]uint32

type Record struct {
	Values [2]uint32
	Count  uint32
}

var Global [2]uint32

func Direct() bool {
	values := [2]uint32{1, 2}
	direct := &values
	view := (*[2]uint32)(values[:])
	return direct == view
}

func Replacement() uint32 {
	values := [2]uint32{1, 2}
	view := (*[2]uint32)(values[:])
	element := &values[0]
	values = [2]uint32{3, 4}
	*view = [2]uint32{5, 6}
	return *element*100 + values[0]*10 + values[1]
}

func Nested() bool {
	value := Record{Values: [2]uint32{1, 2}}
	direct := &value.Values
	view := (*[2]uint32)(value.Values[:])
	return direct == view
}

func Empty() bool {
	values := [0]uint32{}
	direct := &values
	view := (*[0]uint32)(values[:])
	return direct == view
}

func Allocated() bool {
	values := new([2]uint32)
	view := (*[2]uint32)(values[:])
	literal := &[2]uint32{1, 2}
	return values == view && literal == (*[2]uint32)(literal[:])
}

func Named() uint32 {
	values := Pair{1, 2}
	direct := &values
	view := (*Pair)(values[:])
	element := &values[0]
	values = Pair{3, 4}
	if direct != view {
		return 0
	}
	return *element*10 + view[1]
}

func RecordReplacement() uint32 {
	value := Record{Values: [2]uint32{1, 2}, Count: 3}
	view := value.Values[:]
	element := &value.Values[0]
	count := &value.Count
	value = Record{Values: [2]uint32{4, 5}, Count: 6}
	return view[0]*1000 + *element*100 + value.Values[1]*10 + *count
}

func ElementReplacement() uint32 {
	values := [2]Record{{Values: [2]uint32{1, 2}}, {Values: [2]uint32{3, 4}}}
	element := &values[0].Values[1]
	view := values[0].Values[:]
	values[0] = Record{Values: [2]uint32{5, 6}}
	return *element*10 + view[0]
}

func AnonymousReplacement() uint32 {
	value := struct{ Values [2]uint32 }{Values: [2]uint32{1, 2}}
	element := &value.Values[0]
	value = struct{ Values [2]uint32 }{Values: [2]uint32{3, 4}}
	return *element*10 + value.Values[1]
}

func Overlap() uint32 {
	values := []uint32{1, 2, 3}
	left := (*[2]uint32)(values)
	right := (*[2]uint32)(values[1:])
	*right = *left
	return values[0]*100 + values[1]*10 + values[2]
}

func Parallel() uint32 {
	left := [2]uint32{1, 2}
	right := [2]uint32{3, 4}
	leftElement := &left[0]
	rightElement := &right[0]
	left, right = right, left
	return *leftElement*10 + *rightElement
}

func GlobalReplacement() uint32 {
	Global = [2]uint32{1, 2}
	element := &Global[0]
	view := Global[:]
	Global = [2]uint32{3, 4}
	return *element*10 + view[1]
}

func NilCancellation() (panicked bool) {
	defer func() { panicked = recover() != nil }()
	var value *[2]uint32
	_ = &*value
	return
}

func replace[T any](target *T, value T) {
	*target = value
}

func replaceArray[T any](target *[2]T, value [2]T) {
	*target = value
}

func GenericReplacement() uint32 {
	value := [2]uint32{1, 2}
	element := &value[0]
	replace(&value, [2]uint32{3, 4})
	return *element*10 + value[1]
}

func GenericElementReplacement() uint32 {
	values := [2][2]uint32{{1, 2}, {3, 4}}
	element := &values[0][0]
	replaceArray(&values, [2][2]uint32{{5, 6}, {7, 8}})
	return *element*10 + values[0][1]
}

func MapReplacement() uint32 {
	values := map[int][2]uint32{0: {1, 2}}
	copy := values[0]
	element := &copy[0]
	values[0] = [2]uint32{3, 4}
	return *element*10 + values[0][1]
}

func DuplicateElementAddress() bool {
	values := [2]uint32{1, 2}
	view := values[:]
	return &values[0] == &view[0] && &values[0] == &values[0]
}

type Holder[T any] struct{ Value T }

func GenericRecordReplacement() uint32 {
	value := Record{Values: [2]uint32{1, 2}, Count: 3}
	element := &value.Values[0]
	count := &value.Count
	replace(&value, Record{Values: [2]uint32{4, 5}, Count: 6})
	return *element*100 + value.Values[1]*10 + *count
}

func GenericFieldReplacement() uint32 {
	value := Holder[[2]uint32]{Value: [2]uint32{1, 2}}
	element := &value.Value[0]
	value = Holder[[2]uint32]{Value: [2]uint32{3, 4}}
	return *element*10 + value.Value[1]
}

func GenericScalarReplacement() uint32 {
	value := uint32(1)
	replace(&value, uint32(2))
	return value
}

func OrderedReplacement() uint32 {
	value := [2]uint32{1, 2}
	trace := uint32(0)
	left := func() *[2]uint32 {
		trace = trace*10 + 1
		return &value
	}
	right := func() [2]uint32 {
		trace = trace*10 + 2
		return [2]uint32{3, 4}
	}
	*left() = right()
	return trace*100 + value[0]*10 + value[1]
}
