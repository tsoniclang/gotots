package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type PackageRequirements struct {
	profiles []api.IntegerRepresentation
	symbols  map[api.RuntimeSymbol]struct{}
	aliases  []api.PrimitiveAlias
}

func (r PackageRequirements) AllowsProfile(
	profile api.IntegerRepresentation,
) bool {
	return slices.Contains(r.profiles, profile)
}

func (r PackageRequirements) RuntimeSymbols() map[api.RuntimeSymbol]struct{} {
	result := make(map[api.RuntimeSymbol]struct{}, len(r.symbols))
	for symbol := range r.symbols {
		result[symbol] = struct{}{}
	}
	return result
}

func (r PackageRequirements) PrimitiveAliases() []api.PrimitiveAlias {
	return slices.Clone(r.aliases)
}

type requirementEntry struct {
	ID     uint16 `json:"id"`
	Export string `json:"export"`
}

type requirementDocument struct {
	SchemaVersion          uint8              `json:"schemaVersion"`
	IntegerRepresentations []string           `json:"integerRepresentations"`
	PrimitiveAliases       []requirementEntry `json:"primitiveAliases"`
	RuntimeSymbols         []requirementEntry `json:"runtimeSymbols"`
}

func DecodePackageRequirements(data []byte) (PackageRequirements, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var document requirementDocument
	if err := decoder.Decode(&document); err != nil {
		return PackageRequirements{}, &AssemblyError{
			Reason: "decode runtime package requirements: " + err.Error(),
		}
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return PackageRequirements{}, &AssemblyError{
			Reason: "decode runtime package requirements: " + err.Error(),
		}
	}
	if document.SchemaVersion != 1 {
		return PackageRequirements{}, &AssemblyError{
			Reason: fmt.Sprintf(
				"runtime package requirement schema %d is not 1",
				document.SchemaVersion,
			),
		}
	}
	profiles := make([]api.IntegerRepresentation, 0, len(document.IntegerRepresentations))
	for _, spelling := range document.IntegerRepresentations {
		var profile api.IntegerRepresentation
		switch spelling {
		case api.IntegerRepresentationNumber.String():
			profile = api.IntegerRepresentationNumber
		case api.IntegerRepresentationBigInt.String():
			profile = api.IntegerRepresentationBigInt
		default:
			return PackageRequirements{}, &AssemblyError{
				Reason: "runtime package requirement has unknown integer profile " +
					spelling,
			}
		}
		if slices.Contains(profiles, profile) {
			return PackageRequirements{}, &AssemblyError{
				Reason: "runtime package requirement duplicates integer profile " +
					spelling,
			}
		}
		profiles = append(profiles, profile)
	}
	if len(profiles) == 0 {
		return PackageRequirements{}, &AssemblyError{
			Reason: "runtime package requirement has no integer profile",
		}
	}
	aliases := make([]api.PrimitiveAlias, 0, len(document.PrimitiveAliases))
	for _, entry := range document.PrimitiveAliases {
		alias := api.PrimitiveAlias(entry.ID)
		name, err := api.PrimitiveAliasName(alias)
		if err != nil {
			return PackageRequirements{}, err
		}
		if name != entry.Export {
			return PackageRequirements{}, &AssemblyError{
				Reason: fmt.Sprintf(
					"primitive alias %d export is %q, want %q",
					entry.ID,
					entry.Export,
					name,
				),
			}
		}
		if slices.Contains(aliases, alias) {
			return PackageRequirements{}, &AssemblyError{
				Reason: fmt.Sprintf(
					"runtime package requirement duplicates primitive alias %d",
					entry.ID,
				),
			}
		}
		aliases = append(aliases, alias)
	}
	slices.Sort(aliases)
	symbols := make(map[api.RuntimeSymbol]struct{}, len(document.RuntimeSymbols))
	for _, entry := range document.RuntimeSymbols {
		symbol := api.RuntimeSymbol(entry.ID)
		contract, err := api.RuntimeContract(symbol)
		if err != nil {
			return PackageRequirements{}, err
		}
		if contract.ExportedName() != entry.Export {
			return PackageRequirements{}, &AssemblyError{
				Symbol: symbol,
				Reason: fmt.Sprintf(
					"runtime requirement export is %q, want %q",
					entry.Export,
					contract.ExportedName(),
				),
			}
		}
		if _, duplicate := symbols[symbol]; duplicate {
			return PackageRequirements{}, &AssemblyError{
				Symbol: symbol,
				Reason: "runtime package requirement duplicates a symbol",
			}
		}
		symbols[symbol] = struct{}{}
	}
	if len(symbols) == 0 && len(aliases) == 0 {
		return PackageRequirements{}, &AssemblyError{
			Reason: "runtime package requirement surface is empty",
		}
	}
	return PackageRequirements{
		profiles: profiles,
		symbols:  symbols,
		aliases:  aliases,
	}, nil
}
