package catalog

import "fmt"

// SelectedGoVersion is the Go language version this catalog is reconciled
// against. A toolchain upgrade revisits every catalog domain before this
// constant moves.
const SelectedGoVersion = "go1.26"

// Feature is the closed catalog of version-gated language features the
// selected version admits. Values are explicit and permanent.
type Feature uint16

// Explicit, permanent feature identities. Do not renumber; append only.
const (
	FeatureInvalid Feature = 0

	FeatureGenerics            Feature = 1 // type parameters and constraints
	FeatureAnyComparable       Feature = 2 // predeclared any/comparable
	FeatureMinMaxClear         Feature = 3 // built-ins min/max/clear
	FeatureLoopVarPerIteration Feature = 4 // per-iteration loop variables
	FeatureRangeOverInt        Feature = 5 // range over integer
	FeatureRangeOverFunc       Feature = 6 // range over iterator function
	FeatureGenericTypeAlias    Feature = 7 // parameterized type aliases

	// featureCount is the highest assigned identity; append-only.
	featureCount = 7
)

type featureDescriptor struct {
	name       string
	minVersion string // first Go version admitting the feature
}

var featureTable = [featureCount + 1]featureDescriptor{
	FeatureGenerics:            {"generics", "go1.18"},
	FeatureAnyComparable:       {"any-comparable", "go1.18"},
	FeatureMinMaxClear:         {"min-max-clear", "go1.21"},
	FeatureLoopVarPerIteration: {"loopvar-per-iteration", "go1.22"},
	FeatureRangeOverInt:        {"range-over-int", "go1.22"},
	FeatureRangeOverFunc:       {"range-over-func", "go1.23"},
	FeatureGenericTypeAlias:    {"generic-type-alias", "go1.24"},
}

// Valid reports whether f names a feature.
func (f Feature) Valid() bool { return f >= 1 && f <= featureCount }

// Name is the stable descriptive name.
func (f Feature) Name() string {
	if !f.Valid() {
		return ""
	}
	return featureTable[f].name
}

// MinVersion is the first Go version admitting the feature.
func (f Feature) MinVersion() string {
	if !f.Valid() {
		return ""
	}
	return featureTable[f].minVersion
}

// String renders f for reports.
func (f Feature) String() string {
	if name := f.Name(); name != "" {
		return name
	}
	return fmt.Sprintf("catalog.Feature(%d)", uint16(f))
}

// AllFeatures returns every feature in ascending identity order.
func AllFeatures() []Feature {
	out := make([]Feature, 0, featureCount)
	for id := 1; id <= featureCount; id++ {
		out = append(out, Feature(id))
	}
	return out
}
