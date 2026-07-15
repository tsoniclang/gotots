package plan

import "testing"

func TestOwnerOnlySliceSelectsNativeArray(t *testing.T) {
	p := NewPlanner()
	values := p.NewRegion()
	if got := p.SelectSlice(values); got != CandidateNativeArray {
		t.Fatalf("owner-only slice selected %q; want native array", got)
	}
}

func TestRequirementsPropagateAcrossConnections(t *testing.T) {
	p := NewPlanner()
	a := p.NewRegion()
	b := p.NewRegion()
	c := p.NewRegion()
	p.Require(a, ReqNilability)
	p.Connect(a, b)
	p.Connect(b, c)
	if got := p.SelectSlice(c); got != CandidateGoSlice {
		t.Fatalf("connected region ignored nilability: %q", got)
	}
	want := []Requirement{ReqNilability}
	got := p.Requirements(a)
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("requirements = %v; want %v", got, want)
	}
}

func TestConnectionAfterRequirementStillUnions(t *testing.T) {
	p := NewPlanner()
	a := p.NewRegion()
	b := p.NewRegion()
	p.Require(b, ReqSharedView)
	p.Connect(a, b)
	p.Require(a, ReqCapacity)
	got := p.Requirements(b)
	if len(got) != 2 || got[0] != ReqCapacity || got[1] != ReqSharedView {
		t.Fatalf("union lost a requirement: %v", got)
	}
}

func TestEscapeForcesTheExactCarrier(t *testing.T) {
	p := NewPlanner()
	local := p.NewRegion()
	p.Require(local, ReqEscape)
	if got := p.SelectSlice(local); got != CandidateGoSlice {
		t.Fatalf("escaping slice selected %q", got)
	}
}

func TestDeterministicRootUnderAnyMergeOrder(t *testing.T) {
	first := NewPlanner()
	x1 := first.NewRegion()
	y1 := first.NewRegion()
	first.Connect(x1, y1)

	second := NewPlanner()
	x2 := second.NewRegion()
	y2 := second.NewRegion()
	second.Connect(y2, x2)

	if first.find(y1) != second.find(y2) || first.find(x1) != second.find(x2) {
		t.Fatal("merge order changed the canonical region root")
	}
}
