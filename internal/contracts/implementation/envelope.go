package implementation

type EnvelopeKind string

const (
	EnvelopeInvalid           EnvelopeKind = ""
	EnvelopeExact             EnvelopeKind = "exact"
	EnvelopeInternalAlgorithm EnvelopeKind = "internal-algorithm"
)

func (k EnvelopeKind) Valid() bool {
	return k == EnvelopeExact || k == EnvelopeInternalAlgorithm
}

type Envelope struct {
	Kind                 EnvelopeKind `json:"kind"`
	RelaxedBehavior      string       `json:"relaxedBehavior,omitempty"`
	PreservedObservables []string     `json:"preservedObservables,omitempty"`
	Evidence             []string     `json:"evidence,omitempty"`
}

func (e Envelope) Valid() bool {
	if !e.Kind.Valid() {
		return false
	}
	if e.Kind == EnvelopeExact {
		return e.RelaxedBehavior == ""
	}
	return e.RelaxedBehavior != "" &&
		len(e.PreservedObservables) != 0 && len(e.Evidence) != 0
}
