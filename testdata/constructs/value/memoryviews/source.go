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
