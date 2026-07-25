package frontend

// Work is the closed Stage-2 production ledger. Linear counters measure
// operations performed by named passes; output and storage counters are kept
// separate so output cardinality cannot masquerade as construction work.
type Work struct {
	Packages                      int
	InputOccurrences              int
	ChildEdgeAssignments          int
	ContextAssignments            int
	ObjectOccurrenceVisits        int
	CheckerDefinitionVisits       int
	CheckerImplicitEvidenceVisits int
	CheckerSignatureBindingVisits int
	CheckerScopeEvidenceVisits    int
	ImplicitBindingVisits         int
	IntrinsicOccurrenceVisits     int
	CaptureOccurrenceVisits       int
	ResolutionVisits              int
	DefinitionContainmentVisits   int
	DefinitionContainmentEdges    int
	MemberTypeVisits              int
	ContainmentProbes             int
	OccurrenceScopeProbes         int
	CheckerScopeProbes            int
	TypeConstructions             int
	ObjectConstructions           int
	OperationConstructions        int
	OccurrenceResolutions         int
	DefinitionContainmentEntries  int
	CanonicalSortInputs           int
}

func (work *Work) merge(other Work) {
	work.Packages += other.Packages
	work.InputOccurrences += other.InputOccurrences
	work.ChildEdgeAssignments += other.ChildEdgeAssignments
	work.ContextAssignments += other.ContextAssignments
	work.ObjectOccurrenceVisits += other.ObjectOccurrenceVisits
	work.CheckerDefinitionVisits += other.CheckerDefinitionVisits
	work.CheckerImplicitEvidenceVisits +=
		other.CheckerImplicitEvidenceVisits
	work.CheckerSignatureBindingVisits +=
		other.CheckerSignatureBindingVisits
	work.CheckerScopeEvidenceVisits +=
		other.CheckerScopeEvidenceVisits
	work.ImplicitBindingVisits += other.ImplicitBindingVisits
	work.IntrinsicOccurrenceVisits += other.IntrinsicOccurrenceVisits
	work.CaptureOccurrenceVisits += other.CaptureOccurrenceVisits
	work.ResolutionVisits += other.ResolutionVisits
	work.DefinitionContainmentVisits +=
		other.DefinitionContainmentVisits
	work.DefinitionContainmentEdges +=
		other.DefinitionContainmentEdges
	work.MemberTypeVisits += other.MemberTypeVisits
	work.ContainmentProbes += other.ContainmentProbes
	work.OccurrenceScopeProbes += other.OccurrenceScopeProbes
	work.CheckerScopeProbes += other.CheckerScopeProbes
	work.TypeConstructions += other.TypeConstructions
	work.ObjectConstructions += other.ObjectConstructions
	work.OperationConstructions += other.OperationConstructions
	work.OccurrenceResolutions += other.OccurrenceResolutions
	work.DefinitionContainmentEntries +=
		other.DefinitionContainmentEntries
	work.CanonicalSortInputs += other.CanonicalSortInputs
}

func (work Work) Plus(other Work) Work {
	work.merge(other)
	return work
}

func (work Work) LinearOperations() int {
	return work.InputOccurrences +
		work.ChildEdgeAssignments +
		work.ContextAssignments +
		work.ObjectOccurrenceVisits +
		work.CheckerDefinitionVisits +
		work.CheckerImplicitEvidenceVisits +
		work.CheckerSignatureBindingVisits +
		work.CheckerScopeEvidenceVisits +
		work.ImplicitBindingVisits +
		work.IntrinsicOccurrenceVisits +
		work.CaptureOccurrenceVisits +
		work.ResolutionVisits +
		work.DefinitionContainmentVisits +
		work.DefinitionContainmentEdges +
		work.MemberTypeVisits +
		work.ContainmentProbes +
		work.OccurrenceScopeProbes +
		work.CheckerScopeProbes +
		work.TypeConstructions +
		work.ObjectConstructions +
		work.OperationConstructions
}
