package facts

import "testing"

// The seal discipline is the pipeline's ordering guarantee: writes
// after Seal and reads before Seal both panic.
func TestSealDiscipline(t *testing.T) {
	s := New()
	s.PutGenericDemand("p::func::Map", GenericDemand{NeedsEquality: true, BindingFamilies: []string{"b", "a"}})
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("read before Seal must panic")
			}
		}()
		s.GenericDemand("p::func::Map")
	}()
	s.Seal()
	demand, ok := s.GenericDemand("p::func::Map")
	if !ok || !demand.NeedsEquality || demand.BindingFamilies[0] != "a" {
		t.Fatalf("sealed read = %+v %v", demand, ok)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("write after Seal must panic")
			}
		}()
		s.PutGenericDemand("p::func::Other", GenericDemand{})
	}()
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("double Seal must panic")
			}
		}()
		s.Seal()
	}()
}

// One producer per fact: a second write of the same identity panics.
func TestOneProducerPerFact(t *testing.T) {
	s := New()
	s.PutReceiverNilability("p::method::T::M", ReceiverNilability{EquivalentAtEntry: true})
	defer func() {
		if recover() == nil {
			t.Fatal("duplicate fact write must panic")
		}
	}()
	s.PutReceiverNilability("p::method::T::M", ReceiverNilability{})
}
