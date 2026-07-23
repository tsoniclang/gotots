package selectionfacts

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

type definitionEvidence struct {
	pkg      identity.PackageID
	fileHash string
	header   string
	boundary string
	mapping  string
}

type evidenceIndex struct {
	packages    map[identity.PackageID]string
	definitions map[identity.DefinitionID]definitionEvidence
}

func buildEvidenceIndex(
	universe *source.Universe,
	graph *structure.Graph,
) (*evidenceIndex, error) {
	out := &evidenceIndex{
		packages:    map[identity.PackageID]string{},
		definitions: map[identity.DefinitionID]definitionEvidence{},
	}
	fileHashes := map[identity.FileID]string{}
	for _, pkg := range universe.Packages() {
		var packageParts []string
		for _, file := range pkg.Files() {
			fileHashes[file.ID()] = file.ByteDigest().String()
			packageParts = append(
				packageParts,
				file.ID().String()+"="+file.ByteDigest().String(),
			)
		}
		for _, input := range pkg.Inputs() {
			packageParts = append(
				packageParts,
				fmt.Sprintf(
					"%s|%s=%s",
					input.Kind(),
					input.ID(),
					input.ByteDigest(),
				),
			)
		}
		sort.Strings(packageParts)
		out.packages[pkg.ID()] = digest(packageParts...)
	}
	for _, pkg := range graph.Packages() {
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
		for _, definition := range pkg.Definitions() {
			if headers[definition.ID()] == "" ||
				boundaries[definition.ID()] == "" {
				return nil, fmt.Errorf(
					"definition %s lacks structural fact evidence",
					definition.ID(),
				)
			}
			out.definitions[definition.ID()] = definitionEvidence{
				pkg:      pkg.ID(),
				fileHash: fileHashes[definition.ID().File()],
				header:   headers[definition.ID()],
				boundary: boundaries[definition.ID()],
				mapping:  mappings[definition.ID()],
			}
		}
	}
	return out, nil
}

func (i *evidenceIndex) digest(
	definition identity.DefinitionID,
	kind contract.SelectionFactKind,
	value bool,
) (string, error) {
	record, present := i.definitions[definition]
	if !present {
		return "", fmt.Errorf(
			"selection fact %s has no structural evidence", definition,
		)
	}
	packageHash := i.packages[record.pkg]
	if packageHash == "" {
		return "", fmt.Errorf(
			"selection fact %s has no package-byte evidence", definition,
		)
	}
	return digest(
		fmt.Sprintf("selectionfacts-evidence/v%d", SchemaVersion),
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
