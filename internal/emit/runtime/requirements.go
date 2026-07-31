package runtime

import (
	"fmt"
	"slices"

	runtimecontract "github.com/tsoniclang/gotots/internal/contracts/runtime"
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

func ResolvePackageRequirements(
	source runtimecontract.Requirements,
) (PackageRequirements, error) {
	if !source.Valid() {
		return PackageRequirements{}, &AssemblyError{
			Reason: "runtime package requirements are invalid",
		}
	}
	profiles := make([]api.IntegerRepresentation, 0, len(source.Profiles()))
	for _, selected := range source.Profiles() {
		var profile api.IntegerRepresentation
		switch selected {
		case runtimecontract.ProfileNumber:
			profile = api.IntegerRepresentationNumber
		case runtimecontract.ProfileBigInt:
			profile = api.IntegerRepresentationBigInt
		default:
			return PackageRequirements{}, &AssemblyError{
				Reason: "runtime package requirement has an invalid integer profile",
			}
		}
		profiles = append(profiles, profile)
	}
	aliases := make([]api.PrimitiveAlias, 0, len(source.PrimitiveAliases()))
	for _, entry := range source.PrimitiveAliases() {
		alias := api.PrimitiveAlias(entry.ID())
		name, err := api.PrimitiveAliasName(alias)
		if err != nil {
			return PackageRequirements{}, err
		}
		if name != entry.Export() {
			return PackageRequirements{}, &AssemblyError{
				Reason: fmt.Sprintf(
					"primitive alias %d export is %q, want %q",
					entry.ID(),
					entry.Export(),
					name,
				),
			}
		}
		aliases = append(aliases, alias)
	}
	symbols := make(map[api.RuntimeSymbol]struct{}, len(source.RuntimeSymbols()))
	for _, entry := range source.RuntimeSymbols() {
		symbol := api.RuntimeSymbol(entry.ID())
		contract, err := api.RuntimeContract(symbol)
		if err != nil {
			return PackageRequirements{}, err
		}
		if contract.ExportedName() != entry.Export() {
			return PackageRequirements{}, &AssemblyError{
				Symbol: symbol,
				Reason: fmt.Sprintf(
					"runtime requirement export is %q, want %q",
					entry.Export(),
					contract.ExportedName(),
				),
			}
		}
		symbols[symbol] = struct{}{}
	}
	return PackageRequirements{
		profiles: profiles,
		symbols:  symbols,
		aliases:  aliases,
	}, nil
}
