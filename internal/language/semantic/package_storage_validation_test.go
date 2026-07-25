package semantic

import (
	"strings"
	"testing"
)

func TestNormalizedPackageConstructionOwnsArenaConservation(
	t *testing.T,
) {
	tests := []struct {
		name   string
		mutate func(Package) Package
		want   string
	}{
		{
			name: "duplicate-payload-owner",
			mutate: func(pkg Package) Package {
				pkg.definitions.records = append(
					[]storedDefinition(nil),
					pkg.definitions.records...,
				)
				pkg.definitions.records = append(
					pkg.definitions.records,
					pkg.definitions.records[0],
				)
				return pkg
			},
			want: "payload 1 has multiple owners",
		},
		{
			name: "shifted-relation-range",
			mutate: func(pkg Package) Package {
				pkg.definitions.callable = append(
					[]storedCallableDefinition(nil),
					pkg.definitions.callable...,
				)
				pkg.definitions.callable[0].
					declarations.start++
				return pkg
			},
			want: "range 1+1 exceeds 1",
		},
		{
			name: "orphan-payload",
			mutate: func(pkg Package) Package {
				pkg.definitions.callable = append(
					[]storedCallableDefinition(nil),
					pkg.definitions.callable...,
				)
				pkg.definitions.callable = append(
					pkg.definitions.callable,
					storedCallableDefinition{},
				)
				return pkg
			},
			want: "arena index 1 has no owner",
		},
		{
			name: "orphan-relation",
			mutate: func(pkg Package) Package {
				pkg.operations.operands = append(
					[]occurrenceRef(nil),
					pkg.operations.operands...,
				)
				pkg.operations.operands = append(
					pkg.operations.operands,
					pkg.operations.records[0].idOccurrence(
						pkg.identities,
					),
				)
				return pkg
			},
			want: "arena index 0 has no owner",
		},
		{
			name: "noncanonical-record-order",
			mutate: func(pkg Package) Package {
				pkg.types.records = append(
					[]storedType(nil),
					pkg.types.records...,
				)
				pkg.types.records[0],
					pkg.types.records[1] =
					pkg.types.records[1],
					pkg.types.records[0]
				return pkg
			},
			want: "records are not canonical",
		},
		{
			name: "absent-operation-type",
			mutate: func(pkg Package) Package {
				pkg.operations.records = append(
					[]storedOperation(nil),
					pkg.operations.records...,
				)
				pkg.operations.records[0].resultType =
					typeRef(len(pkg.identities.types) + 1)
				return pkg
			},
			want: "absent type",
		},
		{
			name: "missing-operation-resolution",
			mutate: func(pkg Package) Package {
				pkg.resolutions.records = nil
				pkg.resolutions.operations = nil
				return pkg
			},
			want: "1 source operations and 0 operation resolutions",
		},
		{
			name: "duplicate-operation-resolution",
			mutate: func(pkg Package) Package {
				pkg.resolutions.records = append(
					[]storedResolution(nil),
					pkg.resolutions.records...,
				)
				duplicate := pkg.resolutions.records[0]
				pkg.resolutions.operations = append(
					[]operationRef(nil),
					pkg.resolutions.operations...,
				)
				pkg.resolutions.operations = append(
					pkg.resolutions.operations,
					pkg.resolutions.operations[0],
				)
				duplicate.payload = uint64(
					len(pkg.resolutions.operations),
				)
				pkg.resolutions.records = append(
					pkg.resolutions.records,
					duplicate,
				)
				return pkg
			},
			want: "resolution records are not canonical",
		},
		{
			name: "mismatched-operation-resolution-origin",
			mutate: func(pkg Package) Package {
				pkg.identities.operations = append(
					[]storedOperationIdentity(nil),
					pkg.identities.operations...,
				)
				origin := &pkg.identities.operations[0]
				for index := range pkg.identities.occurrences {
					reference := occurrenceRef(index + 1)
					if reference != origin.occurrence {
						origin.occurrence = reference
						break
					}
				}
				return pkg
			},
			want: "operation resolution differs from operation origin",
		},
		{
			name: "invalid-operation-identity-component",
			mutate: func(pkg Package) Package {
				pkg.identities.operations = append(
					[]storedOperationIdentity(nil),
					pkg.identities.operations...,
				)
				pkg.identities.operations[0].definition =
					definitionRef(
						len(pkg.identities.definitions) + 1,
					)
				return pkg
			},
			want: "component references are invalid",
		},
		{
			name: "missing-type-witness",
			mutate: func(pkg Package) Package {
				pkg.witnesses.records = append(
					[]storedTypeWitness(nil),
					pkg.witnesses.records[:1]...,
				)
				return pkg
			},
			want: "types and 1 witnesses",
		},
		{
			name: "orphan-authority",
			mutate: func(pkg Package) Package {
				pkg.authorities.records = append(
					[]Authority(nil),
					pkg.authorities.records...,
				)
				pkg.authorities.records = append(
					pkg.authorities.records,
					pkg.authorities.records[0],
				)
				return pkg
			},
			want: "has no record owner",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pkg := test.mutate(semanticWirePackage(t))
			err := validateNormalizedPackageStorage(pkg)
			if err == nil ||
				!strings.Contains(err.Error(), test.want) {
				t.Fatalf(
					"storage mutation error = %v, want %q",
					err,
					test.want,
				)
			}
		})
	}
}

func (record storedOperation) idOccurrence(
	identities packageIdentityTable,
) occurrenceRef {
	id := identities.operation(record.id)
	return identities.occurrenceReference(id.Occurrence())
}
