package stagecheck

import (
	"fmt"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func missingBindingDetails(
	expected map[identity.SemanticBindingID]*checkerBindingCandidate,
	actual map[identity.SemanticBindingID]semantic.Binding,
) []string {
	var out []string
	for id, candidate := range expected {
		if _, present := actual[id]; present {
			continue
		}
		pkg := "<nil>"
		packageScope := false
		if candidate.object != nil {
			if candidate.object.Pkg() != nil {
				pkg = candidate.object.Pkg().Path()
				packageScope =
					candidate.object.Parent() ==
						candidate.object.Pkg().Scope()
			}
		}
		out = append(out, fmt.Sprintf(
			"%s=%T:%s pkg=%s package-scope=%t",
			id,
			candidate.object,
			types.ObjectString(candidate.object, nil),
			pkg,
			packageScope,
		))
	}
	sort.Strings(out)
	return out
}
