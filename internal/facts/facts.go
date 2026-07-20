// Package facts is the whole-program fact layer of the governing
// pipeline (numbered-order step 17):
//
//	Go AST + go/types -> semantic IR -> SEALED whole-program facts ->
//	immutable total plans -> typed TypeScript AST -> one formatter
//
// Facts are deterministic products of the typed universe, computed to
// a fixed point and sealed BEFORE any planning or lowering decision
// reads them. A sealed store is immutable: every mutation after Seal
// panics — lowering can never supplement or adjust analysis, which is
// the exact failure mode that grew the legacy capability-vector
// protocol. Each fact family has ONE producer; consumers only read.
package facts

import (
	"fmt"
	"sort"
)

// Store holds every fact family for one program. Families are added
// during analysis, then Seal freezes the store for the planning and
// lowering stages.
type Store struct {
	sealed bool

	// genericDemand maps a generic declaration's canonical source
	// identity to the closed operation demand of its binding families
	// (which operations any actual instantiation requires). Producer:
	// the generic evidence closure. This is a SOURCE fact — which Go
	// operations the code performs — never a target-protocol vector.
	genericDemand map[string]GenericDemand

	// receiverNilability maps a method's canonical source identity to
	// its receiver-nilability fact (ADR-0006): whether the first
	// receiver dereference is observationally equivalent to a call-site
	// check, proven per method by the effect analysis.
	receiverNilability map[string]ReceiverNilability
}

// GenericDemand is the closed operation demand of one generic
// declaration across every actual binding family.
type GenericDemand struct {
	// Operations the bindings demand, in Go semantic vocabulary
	// (equality, hashing/map-key, zero construction, value copy,
	// pointer identity), each true only when some actual instantiation
	// requires it.
	NeedsEquality    bool
	NeedsMapKey      bool
	NeedsZero        bool
	NeedsValueCopy   bool
	NeedsPtrIdentity bool
	// BindingFamilies enumerates the distinct actual binding families
	// (ADR-0007 step 4 candidates), sorted, by canonical family key.
	BindingFamilies []string
}

// ReceiverNilability is one method's ADR-0006 proof result.
type ReceiverNilability struct {
	// EquivalentAtEntry: a call-site nil check is observationally
	// equivalent to the first in-body dereference (no effect, call,
	// mutation, or defer boundary precedes it).
	EquivalentAtEntry bool
	// ToleratesNil: the method is semantically defined for nil
	// receivers (calibration class B7) and takes the free-function
	// exception lowering.
	ToleratesNil bool
	// ProvenNonNilSites counts call sites where flow analysis proves
	// the receiver non-nil (check elided entirely).
	ProvenNonNilSites int
}

func New() *Store {
	return &Store{
		genericDemand:      map[string]GenericDemand{},
		receiverNilability: map[string]ReceiverNilability{},
	}
}

func (s *Store) mutable(family string) {
	if s.sealed {
		panic("facts: " + family + " written after Seal — lowering may not supplement analysis")
	}
}

// PutGenericDemand records one declaration's closed demand. Writing
// the same identity twice is a producer defect (one owner per fact).
func (s *Store) PutGenericDemand(sourceID string, demand GenericDemand) {
	s.mutable("genericDemand")
	if _, exists := s.genericDemand[sourceID]; exists {
		panic("facts: generic demand for " + sourceID + " produced twice")
	}
	sort.Strings(demand.BindingFamilies)
	s.genericDemand[sourceID] = demand
}

// PutReceiverNilability records one method's proof result.
func (s *Store) PutReceiverNilability(sourceID string, fact ReceiverNilability) {
	s.mutable("receiverNilability")
	if _, exists := s.receiverNilability[sourceID]; exists {
		panic("facts: receiver nilability for " + sourceID + " produced twice")
	}
	s.receiverNilability[sourceID] = fact
}

// Seal freezes the store. Sealing twice is a pipeline-order defect.
func (s *Store) Seal() {
	if s.sealed {
		panic("facts: sealed twice — the pipeline seals exactly once before planning")
	}
	s.sealed = true
}

// Sealed reports whether planning may begin.
func (s *Store) Sealed() bool { return s.sealed }

// GenericDemand reads one declaration's demand; reading before Seal is
// a pipeline-order defect (planning must never see a moving analysis).
func (s *Store) GenericDemand(sourceID string) (GenericDemand, bool) {
	if !s.sealed {
		panic("facts: read before Seal — planning consumes only sealed facts")
	}
	demand, ok := s.genericDemand[sourceID]
	return demand, ok
}

// ReceiverNilability reads one method's proof result.
func (s *Store) ReceiverNilability(sourceID string) (ReceiverNilability, bool) {
	if !s.sealed {
		panic("facts: read before Seal — planning consumes only sealed facts")
	}
	fact, ok := s.receiverNilability[sourceID]
	return fact, ok
}

// Digest summarizes the sealed store for determinism evidence: sorted
// counts per family (two identical analyses must digest identically).
func (s *Store) Digest() string {
	if !s.sealed {
		panic("facts: digest before Seal")
	}
	return fmt.Sprintf("genericDemand=%d receiverNilability=%d", len(s.genericDemand), len(s.receiverNilability))
}
