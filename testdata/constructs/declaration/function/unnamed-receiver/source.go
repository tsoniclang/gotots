package unnamedreceiver

type Token struct{}

func (Token) Value() int32 {
	return 23
}

func (*Token) Pointer() int32 {
	return 29
}

func Audit() []int32 {
	value := Token{}
	return []int32{value.Value(), value.Pointer()}
}
