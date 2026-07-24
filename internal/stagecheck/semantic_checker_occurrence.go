package stagecheck

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func (verifier *checkerSemanticVerifier) verifyOccurrences() (
	map[identity.SemanticBindingID]map[identity.DefinitionID]bool,
	error,
) {
	captureUses :=
		map[identity.SemanticBindingID]map[identity.DefinitionID]bool{}
	err := verifier.actual.VisitResolutions(func(
		resolution semantic.OccurrenceResolution,
	) error {
		if verifier.localOnly &&
			!verifier.expected.localOccurrence(
				resolution.Occurrence(),
				resolution.Owner(),
			) {
			return nil
		}
		occurrenceReference := verifier.expected.occurrences.reference(
			resolution.Occurrence(),
		)
		if !occurrenceReference.valid() {
			return fmt.Errorf(
				"resolution names absent occurrence %s",
				resolution.Occurrence(),
			)
		}
		occurrence := verifier.expected.occurrenceRecord(
			occurrenceReference,
		)
		occurrenceID := occurrence.ID()
		node, present := verifier.index.OccurrenceNode(occurrenceID)
		if !present {
			return fmt.Errorf(
				"occurrence %s has no transient node", occurrenceID,
			)
		}
		if verifier.index.CheckedUnmapped(occurrenceID) {
			if resolution.Kind() != semantic.ResolutionUnsupported {
				return fmt.Errorf(
					"checked-unmapped occurrence %s is not unsupported",
					occurrenceID,
				)
			}
			return nil
		}
		if resolution.Kind() == semantic.ResolutionStructuralOnly &&
			resolution.Structural().Disposition() ==
				semantic.StructuralIntrinsicContract {
			if resolution.Variant() != catalog.VariantNone {
				return fmt.Errorf(
					"intrinsic occurrence %s carries variant %s",
					occurrenceID, resolution.Variant(),
				)
			}
			return nil
		}
		variant, err := independentSemanticVariant(
			verifier.expected,
			verifier.index,
			occurrence.OccurrenceRef,
			node,
		)
		if err != nil {
			if resolution.Variant() == catalog.VariantNone &&
				resolution.Kind() ==
					semantic.ResolutionStructuralOnly {
				return nil
			}
			return fmt.Errorf(
				"occurrence %s variant: %w", occurrenceID, err,
			)
		}
		if resolution.Variant() != variant {
			return fmt.Errorf(
				"occurrence %s variant=%s, checker=%s",
				occurrenceID, resolution.Variant(), variant,
			)
		}
		if resolution.Kind() == semantic.ResolutionOperation {
			if verifier.independentCompileTimeContext(
				occurrenceReference,
			) {
				return fmt.Errorf(
					"compile-time occurrence %s owns a runtime operation",
					occurrenceID,
				)
			}
			operation, present := verifier.operation(
				resolution.Operation(),
			)
			if !present {
				return fmt.Errorf(
					"occurrence %s operation is absent", occurrenceID,
				)
			}
			if err := verifier.verifyOperation(
				occurrenceReference,
				occurrence.OccurrenceRef,
				node,
				operation,
			); err != nil {
				return err
			}
		} else if err := verifier.verifyResolutionTarget(
			occurrenceReference,
			occurrence.OccurrenceRef,
			resolution,
			node,
		); err != nil {
			return fmt.Errorf(
				"occurrence %s (%s/%s) resolution %s: %w",
				occurrenceID,
				occurrence.Kind(),
				occurrence.Role(),
				resolution.Kind(),
				err,
			)
		}
		return verifier.recordDirectCaptureUse(
			captureUses,
			resolution,
			occurrenceID,
			node,
		)
	})
	return captureUses, err
}
