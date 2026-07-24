package stagecheck

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/selectionfacts"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

type independentDefinitionEvidence struct {
	pkg      identity.PackageID
	fileHash string
	header   string
	boundary string
	mapping  string
}

type independentEvidenceIndex struct {
	packages    map[identity.PackageID]string
	definitions map[identity.DefinitionID]independentDefinitionEvidence
}

func deriveIndependentFactEvidence(
	universe *source.Universe,
	graph *structure.Graph,
) (*independentEvidenceIndex, error) {
	out := &independentEvidenceIndex{
		packages:    map[identity.PackageID]string{},
		definitions: map[identity.DefinitionID]independentDefinitionEvidence{},
	}
	fileHashes := map[identity.FileID]string{}
	for _, pkg := range universe.Packages() {
		var parts []string
		for _, file := range pkg.Files() {
			hash := file.ByteDigest().String()
			fileHashes[file.ID()] = hash
			parts = append(parts, file.ID().String()+"="+hash)
		}
		for _, input := range pkg.Inputs() {
			parts = append(
				parts,
				fmt.Sprintf(
					"%s|%s=%s",
					input.Kind(),
					input.ID(),
					input.ByteDigest(),
				),
			)
		}
		sort.Strings(parts)
		out.packages[pkg.ID()] = independentFactDigest(parts...)
	}
	if err := graph.VisitResidentPackages(func(
		pkg structure.PackageGraph,
	) error {
		headers := map[identity.DefinitionID]string{}
		boundaries := map[identity.DefinitionID]string{}
		mappings := map[identity.DefinitionID]string{}
		for _, header := range pkg.Headers() {
			headers[header.ID().Definition()] = header.Digest()
		}
		for _, boundary := range pkg.Boundaries() {
			boundaries[boundary.ID().Definition()] =
				boundary.CombinedDigest()
		}
		for _, mapping := range pkg.CheckedMappings() {
			mappings[mapping.Definition()] = fmt.Sprintf(
				"%d:%d:%d:%s",
				mapping.OriginLine(),
				mapping.OriginColumn(),
				uint8(mapping.OriginMatch()),
				mapping.CheckedDigest(),
			)
		}
		if err := pkg.VisitDefinitions(func(
			definition structure.ImplementationDefinition,
		) error {
			out.definitions[definition.ID()] =
				independentDefinitionEvidence{
					pkg:      pkg.ID(),
					fileHash: fileHashes[definition.ID().File()],
					header:   headers[definition.ID()],
					boundary: boundaries[definition.ID()],
					mapping:  mappings[definition.ID()],
				}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func (i *independentEvidenceIndex) digest(
	definition identity.DefinitionID,
	kind contract.SelectionFactKind,
	value bool,
) (string, error) {
	record, present := i.definitions[definition]
	if !present {
		return "", fmt.Errorf(
			"independent fact evidence omits %s", definition,
		)
	}
	packageHash := i.packages[record.pkg]
	if packageHash == "" {
		return "", fmt.Errorf(
			"independent package evidence omits %s", record.pkg,
		)
	}
	return independentFactDigest(
		fmt.Sprintf(
			"selectionfacts-evidence/v%d",
			selectionfacts.SchemaVersion,
		),
		definition.String(),
		kind.String(),
		fmt.Sprint(value),
		record.pkg.String(),
		packageHash,
		record.fileHash,
		record.header,
		record.boundary,
		record.mapping,
	), nil
}
