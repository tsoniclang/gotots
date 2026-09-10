package slicedata

import "unsafe"

type Words []uint32

type Record struct {
	Value uint32
}

func NilAndEmpty() int {
	var absent Words
	present := make(Words, 0)
	if unsafe.SliceData(absent) == nil && unsafe.SliceData(present) != nil {
		return 1
	}
	return 0
}

func RetainedCapacity() uint32 {
	backing := Words{1, 2, 3}
	view := backing[1:1]
	pointer := unsafe.SliceData(view)
	if pointer != &backing[1] || pointer != unsafe.SliceData(view) {
		return 0
	}
	*pointer = 7
	backing[1]++
	return *pointer*10 + backing[1]
}

func NilElement() int {
	values := []*uint32{nil}
	pointer := unsafe.SliceData(values)
	if pointer != nil && *pointer == nil {
		value := uint32(9)
		*pointer = &value
		return int(*values[0])
	}
	return 0
}

func RecordAliases() uint32 {
	values := []Record{{Value: 1}, {Value: 2}}
	pointer := unsafe.SliceData(values[1:1])
	pointer.Value = 3
	values[1] = Record{Value: 7}
	return pointer.Value*10 + values[1].Value
}

func ComplexAliases() int {
	values := []complex128{complex(1, 2), complex(3, 4)}
	pointer := unsafe.SliceData(values[1:1])
	*pointer = complex(5, 6)
	values[1] += complex(1, 2)
	return int(real(*pointer))*10 + int(imag(values[1]))
}

func SingleEvaluation() int {
	calls := 0
	values := []uint32{4}
	selectValues := func() []uint32 {
		calls++
		return values
	}
	pointer := unsafe.SliceData(selectValues())
	*pointer++
	return calls*10 + int(values[0])
}
