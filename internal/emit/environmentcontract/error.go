package environmentcontract

import (
	"fmt"
	"go/types"
)

type MemberError struct {
	Owner  *types.TypeName
	Member types.Object
	Cause  error
}

func (e *MemberError) Error() string {
	owner := "<unknown>"
	member := "<unknown>"
	if e.Owner != nil {
		owner = e.Owner.Name()
	}
	if e.Member != nil {
		member = e.Member.Name()
	}
	return fmt.Sprintf(
		"emit environment contract member %q.%q: %v",
		owner,
		member,
		e.Cause,
	)
}

func (e *MemberError) Unwrap() error {
	return e.Cause
}
