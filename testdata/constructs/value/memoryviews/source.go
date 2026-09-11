package memoryviews

import "unsafe"

func ByteString() bool {
	bytes := []byte{0xff, 65}
	text := unsafe.String(&bytes[0], len(bytes))
	return len(text) == 2 && text[0] == 0xff && text[1] == 65
}

func EmptyString() bool {
	return unsafe.String(nil, 0) == ""
}

func RetainedStringLocation() bool {
	bytes := []byte{0xff, 65}
	text := unsafe.String(&bytes[0], len(bytes))
	return unsafe.StringData(text) == &bytes[0] && unsafe.StringData(text[1:]) == &bytes[1]
}

func EmptyPointerView() bool {
	value := uint32(7)
	view := unsafe.Slice(&value, 0)
	return len(view) == 0 && cap(view) == 0 && unsafe.SliceData(view) == &value
}

func StringBeyondSliceLength() bool {
	bytes := []byte{9, 65, 66}
	window := bytes[1:2]
	text := unsafe.String(&window[0], 2)
	return text == "AB" && unsafe.StringData(text[1:]) == &bytes[2]
}

func SliceBackingAlias() bool {
	values := []uint32{3, 4, 5}
	window := values[1:2]
	view := unsafe.Slice(&window[0], 2)
	view[1] = 7
	values[1] = 6
	return len(view) == 2 && cap(view) == 2 && view[0] == 6 && values[2] == 7 && &view[1] == &values[2]
}

func NamedBacking() bool {
	type Bytes []byte
	bytes := Bytes{65, 66}
	text := unsafe.String(&bytes[0], 2)
	return text == "AB" && unsafe.StringData(text) == &bytes[0]
}

func OrderedBacking() bool {
	bytes := []byte{65, 66}
	calls := 0
	get := func() []byte { calls = calls*10 + 1; return bytes }
	index := func() int { calls = calls*10 + 2; return 0 }
	length := func() int { calls = calls*10 + 3; bytes = []byte{88, 89}; return 2 }
	text := unsafe.String(&get()[index()], length())
	return calls == 123 && text == "AB" && bytes[0] == 88
}

func InvalidAddress() (result bool) {
	calls := 0
	defer func() { result = recover() != nil && calls == 0 }()
	bytes := []byte{65}
	index := len(bytes)
	length := func() int { calls++; return 0 }
	_ = unsafe.String(&bytes[index], length())
	return false
}

func EmptyElementView() bool {
	values := []uint32{7, 8}
	view := unsafe.Slice(&values[1], 0)
	return len(view) == 0 && cap(view) == 0 && unsafe.SliceData(view) == &values[1]
}
