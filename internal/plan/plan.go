// Package plan computes representation requirements over closed value-
// flow regions and selects the simplest exact candidate per region —
// the deterministic fixed point of spec chapter 09. Requirements only
// grow (a monotonic product lattice), regions connect through
// assignment, calls, captures, containers, and boundaries, and a
// candidate is chosen only when it satisfies the region's stable set.
package plan

import "sort"

// Requirement is one independent semantic dimension a region's carrier
// must preserve.
type Requirement string

const (
	// ReqNilability: nil is observed (comparison, zero value kept nil).
	ReqNilability Requirement = "nilability"
	// ReqCapacity: cap() or capacity-dependent append aliasing observed.
	ReqCapacity Requirement = "capacity"
	// ReqSharedView: a reslice or array-backed view shares storage.
	ReqSharedView Requirement = "shared-view"
	// ReqEscape: the value crosses a function, field, container,
	// closure, interface, or boundary edge.
	ReqEscape Requirement = "escape"
)

// Region is one closed value-flow region: a set of occurrences whose
// representation must agree.
type Region struct {
	id           int
	requirements map[Requirement]bool
}

// Planner forms regions with union-find and accumulates requirements
// until the fixed point (requirements only grow, so a single pass over
// operations after union completion is stable).
type Planner struct {
	parents []int
	regions []*Region
}

// NewPlanner builds an empty planner.
func NewPlanner() *Planner { return &Planner{} }

// NewRegion introduces one value occurrence's region.
func (p *Planner) NewRegion() int {
	id := len(p.parents)
	p.parents = append(p.parents, id)
	p.regions = append(p.regions, &Region{id: id, requirements: map[Requirement]bool{}})
	return id
}

func (p *Planner) find(id int) int {
	for p.parents[id] != id {
		p.parents[id] = p.parents[p.parents[id]]
		id = p.parents[id]
	}
	return id
}

// Connect merges two occurrences' regions: their representations must
// agree, so their requirement sets union.
func (p *Planner) Connect(a, b int) {
	ra, rb := p.find(a), p.find(b)
	if ra == rb {
		return
	}
	// Deterministic root: the smaller id wins.
	if rb < ra {
		ra, rb = rb, ra
	}
	p.parents[rb] = ra
	for requirement := range p.regions[rb].requirements {
		p.regions[ra].requirements[requirement] = true
	}
	p.regions[rb].requirements = nil
}

// Require adds one requirement to an occurrence's region.
func (p *Planner) Require(id int, requirement Requirement) {
	p.regions[p.find(id)].requirements[requirement] = true
}

// Requirements returns the stable requirement set of an occurrence's
// region in sorted order.
func (p *Planner) Requirements(id int) []Requirement {
	region := p.regions[p.find(id)]
	out := make([]Requirement, 0, len(region.requirements))
	for requirement := range region.requirements {
		out = append(out, requirement)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// SliceCandidate names one reviewed slice-family representation.
type SliceCandidate string

const (
	// CandidateNativeArray: a plain TypeScript array. Satisfies neither
	// nilability (undefined would need a union), capacity observation,
	// nor shared views; the cheapest form for owner-only slices.
	CandidateNativeArray SliceCandidate = "native-array"
	// CandidateGoSlice: the exact backing/offset/length/capacity carrier
	// with undefined for nil. Satisfies every slice requirement.
	CandidateGoSlice SliceCandidate = "goslice-carrier"
)

// SelectSlice returns the deterministic least-cost slice candidate
// satisfying the region's stable requirement set.
func (p *Planner) SelectSlice(id int) SliceCandidate {
	region := p.regions[p.find(id)]
	if len(region.requirements) == 0 {
		return CandidateNativeArray
	}
	// Nilability, capacity, shared views, and escapes (whose far side
	// may observe any of those) all need the exact carrier.
	return CandidateGoSlice
}
