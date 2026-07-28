package mapruntime

import "fmt"

type Member uint8

const (
	MemberInvalid  Member = 0
	MemberNil      Member = 1
	MemberMake     Member = 2
	MemberLookup   Member = 3
	MemberLookupOK Member = 4
	MemberStore    Member = 5
	MemberDelete   Member = 6
	MemberLength   Member = 7
	MemberIsNil    Member = 8
	MemberClear    Member = 9
)

func Name(member Member) (string, error) {
	switch member {
	case MemberNil:
		return "nil", nil
	case MemberMake:
		return "make", nil
	case MemberLookup:
		return "lookup", nil
	case MemberLookupOK:
		return "lookupOk", nil
	case MemberStore:
		return "store", nil
	case MemberDelete:
		return "delete", nil
	case MemberLength:
		return "length", nil
	case MemberIsNil:
		return "isNil", nil
	case MemberClear:
		return "clear", nil
	default:
		return "", &MemberError{Member: member}
	}
}

type MemberError struct {
	Member Member
}

func (e *MemberError) Error() string {
	return fmt.Sprintf("map runtime member %d is invalid", e.Member)
}
