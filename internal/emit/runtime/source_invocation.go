package runtime

import "slices"

type InvocationContract struct {
	sourceIdentity         string
	sourcePath             string
	exportedName           string
	exactImplementation    bool
	inputParameters        []uint32
	resultOriginParameters []uint32
}

func (p Package) InvocationContracts() []InvocationContract {
	return slices.Clone(p.invocationContracts)
}

func (c InvocationContract) SourceIdentity() string {
	return c.sourceIdentity
}

func (c InvocationContract) SourcePath() string {
	return c.sourcePath
}

func (c InvocationContract) ExportedName() string {
	return c.exportedName
}

func (c InvocationContract) ExactImplementation() bool {
	return c.exactImplementation
}

func (c InvocationContract) InputParameters() []uint32 {
	return slices.Clone(c.inputParameters)
}

func (c InvocationContract) ResultOriginParameters() []uint32 {
	return slices.Clone(c.resultOriginParameters)
}
