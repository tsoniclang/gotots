package environment

type ByteOrder uint8

const (
	ByteOrderInvalid ByteOrder = iota
	ByteOrderLittleEndian
	ByteOrderBigEndian
)

func (o ByteOrder) Valid() bool {
	return o == ByteOrderLittleEndian || o == ByteOrderBigEndian
}

func (p BuildProfile) ByteOrder() (ByteOrder, error) {
	switch p.goarch {
	case "386", "amd64", "arm", "arm64", "loong64", "mipsle",
		"mips64le", "ppc64le", "riscv64", "wasm":
		return ByteOrderLittleEndian, nil
	case "mips", "mips64", "ppc64", "s390x":
		return ByteOrderBigEndian, nil
	default:
		return ByteOrderInvalid, &BuildProfileError{
			Field:  "GOARCH",
			Reason: "byte order is unknown",
		}
	}
}
