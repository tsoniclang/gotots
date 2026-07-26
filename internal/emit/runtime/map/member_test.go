package mapruntime

import "testing"

func TestMemberIdentitiesAndNamesArePinned(t *testing.T) {
	for _, test := range []struct {
		member Member
		id     uint8
		name   string
	}{
		{MemberNil, 1, "nil"},
		{MemberMake, 2, "make"},
		{MemberLookup, 3, "lookup"},
		{MemberLookupOK, 4, "lookupOk"},
		{MemberStore, 5, "store"},
		{MemberDelete, 6, "delete"},
		{MemberLength, 7, "length"},
		{MemberIsNil, 8, "isNil"},
	} {
		if uint8(test.member) != test.id {
			t.Fatalf("%q identity = %d, want %d", test.name, test.member, test.id)
		}
		name, err := Name(test.member)
		if err != nil {
			t.Fatal(err)
		}
		if name != test.name {
			t.Fatalf("member %d name = %q, want %q", test.member, name, test.name)
		}
	}
}

func TestMemberOwnerRejectsInvalidIdentity(t *testing.T) {
	for _, member := range []Member{MemberInvalid, Member(9), Member(255)} {
		if _, err := Name(member); err == nil {
			t.Fatalf("member %d was accepted", member)
		}
	}
}
