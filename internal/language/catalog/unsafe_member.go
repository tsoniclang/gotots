package catalog

import "fmt"

type UnsafeMemberClass uint8

const (
	UnsafeMemberClassInvalid UnsafeMemberClass = iota
	UnsafeMemberClassType
	UnsafeMemberClassBuiltin
)

func (class UnsafeMemberClass) Valid() bool {
	return class >= UnsafeMemberClassType &&
		class <= UnsafeMemberClassBuiltin
}

type UnsafeMemberKind uint8

const (
	UnsafeMemberInvalid UnsafeMemberKind = 0

	UnsafeMemberPointer    UnsafeMemberKind = 1
	UnsafeMemberAlignof    UnsafeMemberKind = 2
	UnsafeMemberOffsetof   UnsafeMemberKind = 3
	UnsafeMemberSizeof     UnsafeMemberKind = 4
	UnsafeMemberAdd        UnsafeMemberKind = 5
	UnsafeMemberSlice      UnsafeMemberKind = 6
	UnsafeMemberSliceData  UnsafeMemberKind = 7
	UnsafeMemberString     UnsafeMemberKind = 8
	UnsafeMemberStringData UnsafeMemberKind = 9

	unsafeMemberCount = UnsafeMemberStringData
)

type unsafeMemberSpec struct {
	name  string
	class UnsafeMemberClass
}

var unsafeMemberSpecs = [unsafeMemberCount + 1]unsafeMemberSpec{
	UnsafeMemberPointer:    {"Pointer", UnsafeMemberClassType},
	UnsafeMemberAlignof:    {"Alignof", UnsafeMemberClassBuiltin},
	UnsafeMemberOffsetof:   {"Offsetof", UnsafeMemberClassBuiltin},
	UnsafeMemberSizeof:     {"Sizeof", UnsafeMemberClassBuiltin},
	UnsafeMemberAdd:        {"Add", UnsafeMemberClassBuiltin},
	UnsafeMemberSlice:      {"Slice", UnsafeMemberClassBuiltin},
	UnsafeMemberSliceData:  {"SliceData", UnsafeMemberClassBuiltin},
	UnsafeMemberString:     {"String", UnsafeMemberClassBuiltin},
	UnsafeMemberStringData: {"StringData", UnsafeMemberClassBuiltin},
}

func (kind UnsafeMemberKind) Valid() bool {
	return kind > UnsafeMemberInvalid && kind <= unsafeMemberCount
}

func (kind UnsafeMemberKind) Name() string {
	if kind.Valid() {
		return unsafeMemberSpecs[kind].name
	}
	return ""
}

func (kind UnsafeMemberKind) Class() UnsafeMemberClass {
	if kind.Valid() {
		return unsafeMemberSpecs[kind].class
	}
	return UnsafeMemberClassInvalid
}

func (kind UnsafeMemberKind) String() string {
	if name := kind.Name(); name != "" {
		return name
	}
	return fmt.Sprintf(
		"catalog.UnsafeMemberKind(%d)", uint8(kind),
	)
}

func UnsafeMemberByName(name string) UnsafeMemberKind {
	for kind := UnsafeMemberKind(1); kind <= unsafeMemberCount; kind++ {
		if kind.Name() == name {
			return kind
		}
	}
	return UnsafeMemberInvalid
}

func AllUnsafeMembers() []UnsafeMemberKind {
	out := make([]UnsafeMemberKind, 0, unsafeMemberCount)
	for kind := UnsafeMemberKind(1); kind <= unsafeMemberCount; kind++ {
		out = append(out, kind)
	}
	return out
}
