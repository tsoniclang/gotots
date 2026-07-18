package ir

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
	Code string `json:"code"`
	// Kind is the producer-owned closed classification; Class is its stable
	// string key (Kind.String()), kept for JSON/histogram display. The
	// inventory dispositions by Kind, never by re-parsing Construct.
	Kind      UnsupportedKind `json:"-"`
	Class     string          `json:"class"`
	Construct string          `json:"construct"`
	Span      Span            `json:"span"`
}

// recordSite converts one Unsupported into a site record; every other
// error is infrastructural and propagates.
func (b *builder) recordSite(err error) bool {
	unsupported, ok := AsUnsupported(err)
	if !ok {
		return false
	}
	*b.sites = append(*b.sites, SiteOf(unsupported))
	return true
}

// SiteOf projects one Unsupported diagnostic to its site record, carrying
// the producer-owned Kind and its stable class key.
func SiteOf(unsupported *Unsupported) UnsupportedSite {
	return UnsupportedSite{
		Code:      unsupported.Code,
		Kind:      unsupported.Kind,
		Class:     unsupported.Kind.String(),
		Construct: unsupported.Construct,
		Span:      unsupported.Span,
	}
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

// UnimplementedStmt marks one statement whose semantic class has no
// accepted lowering. It exists only inside unimplemented bodies; the
// emitter never prints it, because unimplemented bodies emit nothing.
type UnimplementedStmt struct {
	Site UnsupportedSite
}

func (*UnimplementedStmt) stmt() {}
