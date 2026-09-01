package sourcefact

import "fmt"

type Error struct {
	Subject string
	Reason  string
}

func (e *Error) Error() string {
	if e.Subject == "" {
		return "emit canonical Go source fact: " + e.Reason
	}
	return fmt.Sprintf(
		"emit canonical Go source fact for %q: %s",
		e.Subject,
		e.Reason,
	)
}
