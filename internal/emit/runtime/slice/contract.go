package slice

type Member uint8

const (
	MemberInvalid Member = iota
	MemberNil
	MemberMake
	MemberLiteral
	MemberIsNil
	MemberGet
	MemberSet
	MemberSlice
	MemberAppend
	MemberCopy
	MemberAddress
	MemberLength
	MemberCapacity
	MemberAppendSlice
	MemberClear
	MemberArrayLocation
)

const (
	StorageAllocateMember      = "$allocate"
	StorageGrownCapacityMember = "$grownCapacity"
	StorageInitializeMember    = "$initialize"
	StorageWithLengthMember    = "$withLength"
)

func MemberName(member Member) string {
	switch member {
	case MemberNil:
		return "nil"
	case MemberMake:
		return "make"
	case MemberLiteral:
		return "literal"
	case MemberIsNil:
		return "isNil"
	case MemberGet:
		return "get"
	case MemberSet:
		return "set"
	case MemberSlice:
		return "slice"
	case MemberAppend:
		return "append"
	case MemberCopy:
		return "copy"
	case MemberAddress:
		return "address"
	case MemberLength:
		return "length"
	case MemberCapacity:
		return "capacity"
	case MemberAppendSlice:
		return "appendSlice"
	case MemberClear:
		return "clear"
	case MemberArrayLocation:
		return "$arrayLocation"
	default:
		panic("invalid RuntimeSlice member")
	}
}
