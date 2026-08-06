package genericbuiltins

func appendBytes[Bytes ~[]byte | ~string](dst []byte, src Bytes) []byte {
	return append(dst, src...)
}

func BytesResult() string {
	return string(appendBytes([]byte{'<'}, []byte("bytes>")))
}

func StringResult() string {
	return string(appendBytes([]byte{'<'}, "string>"))
}
