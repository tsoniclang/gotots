package embeddedinterface

type Reader interface {
	Read() int32
}

type ValueReader struct {
	Value int32
}

func (reader *ValueReader) Read() int32 {
	return reader.Value
}

type Holder struct {
	Reader
}

func Audit() int32 {
	holder := &Holder{
		Reader: &ValueReader{Value: 41},
	}
	var selected Reader = holder
	return selected.Read()
}
