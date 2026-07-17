package ir

import "strings"

// SupportState is the implementation support of one selected body.
type SupportState string

const (
	// SupportGenerated: the complete body lowered through the reviewed
	// subset.
	SupportGenerated SupportState = "generated"
	// SupportUnimplemented: at least one semantic operation has no
	// accepted lowering; every such site is recorded and no runnable
	// body is emitted.
	SupportUnimplemented SupportState = "unimplemented"
)

// UnsupportedSite is one exact operation outside the reviewed subset,
// recorded without stopping IR construction of the rest of the body.
type UnsupportedSite struct {
	Code      string `json:"code"`
	Class     string `json:"class"`
	Construct string `json:"construct"`
	Span      Span   `json:"span"`
}

// recordSite converts one Unsupported into a site record; every other
// error is infrastructural and propagates.
func (b *builder) recordSite(err error) bool {
	unsupported, ok := AsUnsupported(err)
	if !ok {
		return false
	}
	*b.sites = append(*b.sites, UnsupportedSite{
		Code:      unsupported.Code,
		Class:     ClassOf(unsupported),
		Construct: unsupported.Construct,
		Span:      unsupported.Span,
	})
	return true
}

// AsUnsupported unwraps an error chain to its Unsupported diagnostic.
func AsUnsupported(err error) (*Unsupported, bool) {
	for err != nil {
		if unsupported, ok := err.(*Unsupported); ok {
			return unsupported, true
		}
		unwrapper, ok := err.(interface{ Unwrap() error })
		if !ok {
			return nil, false
		}
		err = unwrapper.Unwrap()
	}
	return nil, false
}

// classPayloadPrefixes are the construct spellings that carry a per-site
// payload after a stable class prefix; ClassOf strips the payload so a
// class key names the semantic family, never a source location or type
// spelling.
var classPayloadPrefixes = []string{
	"non-basic type ", "basic type ", "pointer to non-named type ",
	"pointer to non-struct type ", "pointer to type outside the translated unit: ",
	"interface type ", "array type ", "channel type ", "type parameter ",
	"call outside the translated unit ", "method call outside the translated unit ",
	"map key type ", "type ", "identifier ", "call of ", "field access on ",
	"index on ", "nil comparison on ", "operator ", "conversion from ",
	"non-field selector ", "method value outside the translated unit (",
	"method expression outside the translated unit (", "pointer-receiver method value on ",
	"len of ", "cap of ", "make of ", "builtin ", "constant of type ", "zero value of ",
	"nil of type ", "equality on ", "ordering on ", "inc/dec of ",
	"struct type ", "composite literal of ", "reslice of ", "range over ",
	"dereference of ", "address of ", "method call on ", "clear of ",
	"panic with ", "switch tag of ", "switch case of ", "type assertion on ",
	"type switch on ", "append to ", "copy between ", "interface value of type ",
	"generic function instantiated with an unreviewed type argument ",
	"expression statement ", "assignment to ", "compound assignment to ",
	"indexed compound assignment on ", "field assignment on ", "branch ",
	"package-level ", "string constant with ",
}

// ClassConstructKeys returns the closed set of construct keys a normalized
// class can carry: the trimmed spelling of every registered payload prefix.
// A class's construct portion (ClassOf's "CODE: <construct>") is exactly
// one of these when the diagnostic matched a prefix; the inventory's site
// classifier is a total function over this set, so no key is silently
// defaulted.
func ClassConstructKeys() []string {
	keys := make([]string, len(classPayloadPrefixes))
	for i, prefix := range classPayloadPrefixes {
		keys[i] = strings.TrimSpace(prefix)
	}
	return keys
}

// ClassOf normalizes one diagnostic to its semantic-class key.
func ClassOf(unsupported *Unsupported) string {
	construct := unsupported.Construct
	for _, prefix := range classPayloadPrefixes {
		if strings.HasPrefix(construct, prefix) {
			construct = strings.TrimSpace(prefix)
			break
		}
	}
	return unsupported.Code + ": " + construct
}

// UnimplementedStmt marks one statement whose semantic class has no
// accepted lowering. It exists only inside unimplemented bodies; the
// emitter never prints it, because unimplemented bodies emit nothing.
type UnimplementedStmt struct {
	Site UnsupportedSite
}

func (*UnimplementedStmt) stmt() {}
