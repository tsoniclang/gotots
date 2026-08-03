package naming

import (
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
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

func (n *File) UnsafeCodec(sourceType types.Type) (api.NameReference, error) {
	if sourceType == nil || api.ContainsGenericTypeParameter(sourceType) {
		return api.NameReference{}, &api.NameError{
			Reason: "unsafe-codec identity is invalid",
		}
	}
	artifactKey, err := typeidentity.BuildKey(
		sourceType,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, err := n.owner.registry.internUnsafeCodec(
		artifactKey,
		sourceType,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	request, err := api.NewUnsafeCodecRequest(binding.owner)
	if err != nil {
		return api.NameReference{}, err
	}
	return n.generatedValueReference(
		binding.owner,
		binding.name,
		request,
		api.ArtifactFacetValueSurface,
	)
}

func (r *Registry) internUnsafeCodec(
	artifactKey string,
	sourceType types.Type,
) (unsafeCodecBinding, error) {
	if r == nil || artifactKey == "" || sourceType == nil ||
		api.ContainsGenericTypeParameter(sourceType) {
		return unsafeCodecBinding{}, &api.NameError{
			Reason: "unsafe-codec canonicalization input is invalid",
		}
	}
	if existing, ok := r.unsafeCodecs[artifactKey]; ok {
		bound, valid := existing.owner.UnsafeCodecType()
		if !valid || !types.Identical(bound, sourceType) {
			return unsafeCodecBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "unsafe-codec key joined non-identical Go types",
			}
		}
		return existing, nil
	}
	name, err := interfaceTargetName("$goUnsafeCodec_", artifactKey)
	if err != nil {
		return unsafeCodecBinding{}, err
	}
	if err := reserveGeneratedName(
		r.unsafeCodecNames,
		name,
		artifactKey,
		"unsafe codec",
	); err != nil {
		return unsafeCodecBinding{}, err
	}
	owner, err := api.NewCompilationGeneratedArtifact(
		api.GeneratedArtifactUnsafeCodec,
		sourceType,
		artifactKey,
		name,
		output.UnsafeCodecSupportPath,
	)
	if err != nil {
		return unsafeCodecBinding{}, err
	}
	binding := unsafeCodecBinding{owner: owner, name: name}
	r.unsafeCodecs[artifactKey] = binding
	return binding, nil
}
