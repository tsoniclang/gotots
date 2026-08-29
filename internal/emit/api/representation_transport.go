package api

type GeneratedRepresentationTransportKind uint8

const (
	GeneratedRepresentationTransportInvalid GeneratedRepresentationTransportKind = iota
	GeneratedRepresentationTransportFunctionKernel
	GeneratedRepresentationTransportMemberKernel
)

func (k GeneratedRepresentationTransportKind) Valid() bool {
	return k == GeneratedRepresentationTransportFunctionKernel ||
		k == GeneratedRepresentationTransportMemberKernel
}

type GeneratedRepresentationTransport struct {
	kind       GeneratedRepresentationTransportKind
	sourcePath string
	exportName string
	memberName string
}

func NewGeneratedRepresentationTransport(
	kind GeneratedRepresentationTransportKind,
	sourcePath string,
	exportName string,
	memberName string,
) (GeneratedRepresentationTransport, error) {
	validShape := kind == GeneratedRepresentationTransportFunctionKernel &&
		memberName == "" ||
		kind == GeneratedRepresentationTransportMemberKernel && memberName != ""
	if !kind.Valid() || sourcePath == "" || exportName == "" || !validShape {
		return GeneratedRepresentationTransport{}, &InvariantError{
			Reason: "generated representation transport identity is invalid",
		}
	}
	return GeneratedRepresentationTransport{
		kind:       kind,
		sourcePath: sourcePath,
		exportName: exportName,
		memberName: memberName,
	}, nil
}

func (t GeneratedRepresentationTransport) Valid() bool {
	_, err := NewGeneratedRepresentationTransport(
		t.kind,
		t.sourcePath,
		t.exportName,
		t.memberName,
	)
	return err == nil
}

func (t GeneratedRepresentationTransport) Kind() GeneratedRepresentationTransportKind {
	return t.kind
}

func (t GeneratedRepresentationTransport) SourcePath() string {
	return t.sourcePath
}

func (t GeneratedRepresentationTransport) ExportName() string {
	return t.exportName
}

func (t GeneratedRepresentationTransport) MemberName() string {
	return t.memberName
}

func (t GeneratedRepresentationTransport) Key() string {
	return string(rune(t.kind)) + "\x00" + t.sourcePath + "\x00" +
		t.exportName + "\x00" + t.memberName
}
