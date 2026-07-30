package stringvalues

const packageBytes string = "\xff"
const hexDigits = "0123456789ABCDEF"

var PackageValue string

type Offset int

type Path string

func ASCII() string {
	return "Go"
}

func UTF8() string {
	return "é"
}

func RawUTF8() string {
	return `é`
}

func InvalidBytes() string {
	return "\xff\x00A"
}

func Constants() string {
	const localBytes string = "\x80"
	return packageBytes + localBytes
}

func Zero() string {
	var value string
	return value
}

func Assign(value string) string {
	var result string
	result = value
	return result
}

func PackageZero() string {
	return PackageValue
}

func PackageAssign(value string) string {
	PackageValue = value
	return PackageValue
}

func Concat(left string, right string) string {
	return left + right
}

func Equal(left string, right string) bool {
	return left == right
}

func NotEqual(left string, right string) bool {
	return left != right
}

func Less(left string, right string) bool {
	return left < right
}

func LessEqual(left string, right string) bool {
	return left <= right
}

func Greater(left string, right string) bool {
	return left > right
}

func GreaterEqual(left string, right string) bool {
	return left >= right
}

func Length(value string) int {
	return len(value)
}

func ByteAt(value string, index int) byte {
	return value[index]
}

func ConstantByteAt(index int) byte {
	return hexDigits[index]
}

func Window(value string, low int, high int) string {
	return value[low:high]
}

func Prefix(value string, high int) string {
	return value[:high]
}

func Suffix(value string, low int) string {
	return value[low:]
}

func SuffixCall(low int) (string, int32) {
	var calls int32
	next := func() string {
		calls++
		return "ab"
	}
	return next()[low:], calls
}

func IndexCall(index int) (byte, int32) {
	var calls int32
	next := func() string {
		calls++
		return "ab"
	}
	return next()[index], calls
}

func DefinedBounds() (byte, string) {
	value := "abc"
	low := Offset(1)
	high := Offset(3)
	return value[low], value[low:high]
}

func DefinedWindow(value Path, low int, high int) Path {
	return value[low:high]
}
