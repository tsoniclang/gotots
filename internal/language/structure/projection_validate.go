package structure

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

// Validate enforces resident-index integrity, exact manifest projection, and
// the bounded logical census. Detailed certified packages are independently
// validated one at a time by stagecheck.
func Validate(graph *Graph) error {
	if graph == nil || graph.version != ArtifactVersion {
		return fmt.Errorf("invalid structural artifact version")
	}
	if err := validateProjectionPlan(graph); err != nil {
		return err
	}
	if err := validateCompleteGraph(graph); err != nil {
		return fmt.Errorf("resident structural graph: %w", err)
	}
	return validateDefinitionCensus(graph)
}

func validateProjectionPlan(graph *Graph) error {
	if len(graph.packages) != len(graph.projections) {
		return fmt.Errorf(
			"structural package/projection cardinality is %d/%d",
			len(graph.packages),
			len(graph.projections),
		)
	}
	var previous identity.PackageID
	for index, projection := range graph.projections {
		resident := graph.packages[index]
		if projection.id.IsZero() ||
			resident.id != projection.id ||
			(!previous.IsZero() &&
				previous.Compare(projection.id) >= 0) {
			return fmt.Errorf(
				"noncanonical package projection at %s", projection.id,
			)
		}
		previous = projection.id
		residentFiles := map[identity.FileID]bool{}
		for _, file := range resident.files {
			residentFiles[file.owner.id.file] = true
		}
		certifiedFiles := map[identity.FileID]bool{}
		for _, file := range projection.certifiedFiles {
			if file.id.IsZero() ||
				!validSHA256(file.byteDigest) ||
				residentFiles[file.id] ||
				certifiedFiles[file.id] {
				return fmt.Errorf(
					"invalid certified file projection %s", file.id,
				)
			}
			certifiedFiles[file.id] = true
			if graph.provider == nil ||
				!graph.provider.HasPackageFile(projection.id, file.id) {
				return fmt.Errorf(
					"certified file projection %s lacks manifest authority",
					file.id,
				)
			}
		}
		manifestFileCount := 0
		if graph.provider != nil {
			manifestFileCount = graph.provider.PackageFileCount(
				projection.id,
			)
		}
		if manifestFileCount != len(certifiedFiles) {
			return fmt.Errorf(
				"certified file projection for %s is not an exact manifest set",
				projection.id,
			)
		}
		if graph.provider != nil &&
			graph.provider.HasSyntheticPackage(projection.id) !=
				projection.certifiedSynthetic {
			return fmt.Errorf(
				"certified synthetic projection %s disagrees with manifest",
				projection.id,
			)
		}
	}
	return nil
}

func validateCompletePackage(pkg PackageGraph) error {
	graph := &Graph{
		version:  ArtifactVersion,
		packages: []PackageGraph{pkg},
	}
	if err := sealGraph(graph); err != nil {
		return err
	}
	return validateCompleteGraph(graph)
}
