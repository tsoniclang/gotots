package naming

import (
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/output"
)

func (n *File) Primitive(alias api.PrimitiveAlias) (api.NameReference, error) {
	if existing := n.primitives[alias]; existing != "" {
		modulePath, err := output.ModuleSpecifier(
			n.targetPath,
			output.ScalarSupportPath,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		request, err := api.NewPrimitiveAliasRequest(
			n.factory,
			modulePath,
			alias,
			existing,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		return api.NewNameReference(existing, request)
	}
	exportedName, err := api.PrimitiveAliasName(alias)
	if err != nil {
		return api.NameReference{}, err
	}
	localName := exportedName
	if n.sourceNameExists(localName) ||
		n.hasImportName(localName) {
		base := exportedName + "__from_gotots_support"
		localName = base
		for suffix := uint64(1); n.sourceNameExists(localName) ||
			n.hasImportName(localName); suffix++ {
			localName = base + "_" + strconv.FormatUint(suffix, 10)
		}
	}
	n.importNames[localName] = struct{}{}
	n.primitives[alias] = localName
	modulePath, err := output.ModuleSpecifier(
		n.targetPath,
		output.ScalarSupportPath,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	request, err := api.NewPrimitiveAliasRequest(
		n.factory,
		modulePath,
		alias,
		localName,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewNameReference(localName, request)
}

func (n *File) Runtime(
	symbol api.RuntimeSymbol,
	phase api.ImportPhase,
) (api.NameReference, error) {
	contract, err := api.RuntimeContract(symbol)
	if err != nil {
		return api.NameReference{}, err
	}
	modulePath, err := output.ModuleSpecifier(
		n.targetPath,
		contract.OutputPath(),
	)
	if err != nil {
		return api.NameReference{}, err
	}
	if existing := n.runtime[symbol]; existing != "" {
		request, err := api.NewRuntimeImportRequest(
			n.factory,
			phase,
			modulePath,
			symbol,
			existing,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		return api.NewNameReference(existing, request)
	}
	exportedName := contract.ExportedName()
	localName := exportedName
	if n.sourceNameExists(localName) ||
		n.hasImportName(localName) {
		base := exportedName + "__from_gotots_runtime"
		localName = base
		for suffix := uint64(1); n.sourceNameExists(localName) ||
			n.hasImportName(localName); suffix++ {
			localName = base + "_" + strconv.FormatUint(suffix, 10)
		}
	}
	n.importNames[localName] = struct{}{}
	n.runtime[symbol] = localName
	request, err := api.NewRuntimeImportRequest(
		n.factory,
		phase,
		modulePath,
		symbol,
		localName,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewNameReference(localName, request)
}
