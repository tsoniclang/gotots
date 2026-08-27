package callableimplementation

import (
	"go/ast"
	"go/types"

	callableimplementationcontract "github.com/tsoniclang/gotots/internal/contracts/callableimplementation"
	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/load"
)

type sourceCallable struct {
	function   *types.Func
	signature  string
	bodyDigest string
}

func (p *Prepared) Join(program *load.Program) (*Certificate, error) {
	if p == nil || program == nil || !p.buildProfile.Valid() ||
		!p.compilation.valid() || len(p.modules) == 0 || p.digest == "" {
		return nil, &Error{Operation: "join", Reason: "prepared evidence or selected program is invalid"}
	}
	if !sameBuildProfile(p.buildProfile, program.BuildProfile()) {
		return nil, &Error{Operation: "join build profile", Reason: "selected profile differs"}
	}
	certificate := &Certificate{
		buildProfile: p.buildProfile,
		compilation:  p.compilation,
		byFunction:   make(map[*types.Func]Implementation),
		byIdentity:   make(map[string]Implementation),
		modules:      slicesCloneModules(p.modules),
		digest:       p.digest,
	}
	for _, module := range p.modules {
		selectedPackage := program.PackageByPath(module.packagePath)
		if selectedPackage == nil || selectedPackage.Kind() != load.PackageSource ||
			selectedPackage.ModulePath() != module.modulePath ||
			selectedPackage.ModuleVersion() != module.moduleVersion {
			return nil, &Error{
				Operation: "join package", Subject: module.packagePath,
				Reason: "selected source package identity differs",
			}
		}
		callables, err := sourceCallables(selectedPackage)
		if err != nil {
			return nil, err
		}
		for _, claim := range module.callableClaims {
			selected, ok := callables[claim.SourceIdentity]
			if !ok {
				return nil, &Error{
					Operation: "join callable", Subject: claim.SourceIdentity,
					Reason: "selected source declaration is absent or bodyless",
				}
			}
			if selected.signature != claim.SourceSignature {
				return nil, &Error{
					Operation: "join callable", Subject: claim.SourceIdentity,
					Reason: "selected source signature differs",
				}
			}
			if selected.bodyDigest != claim.SourceBodyDigest {
				return nil, &Error{
					Operation: "join callable", Subject: claim.SourceIdentity,
					Reason: "selected source body differs",
				}
			}
			implementation := Implementation{
				function:         selected.function,
				sourceIdentity:   claim.SourceIdentity,
				sourceSignature:  claim.SourceSignature,
				sourceBodyDigest: claim.SourceBodyDigest,
				variant:          claim.Variant,
				export:           claim.Export,
				module:           module,
			}
			certificate.byFunction[selected.function] = implementation
			certificate.byIdentity[claim.SourceIdentity] = implementation
		}
	}
	if !certificate.Valid() {
		return nil, &Error{Operation: "join", Reason: "certificate is incomplete"}
	}
	return certificate, nil
}

func sourceCallables(sourcePackage *load.Package) (map[string]sourceCallable, error) {
	result := make(map[string]sourceCallable)
	for _, file := range sourcePackage.Files() {
		for _, declaration := range file.Syntax().Decls {
			functionDeclaration, ok := declaration.(*ast.FuncDecl)
			if !ok || functionDeclaration.Body == nil ||
				functionDeclaration.Name.Name == "init" || functionDeclaration.Name.Name == "_" {
				continue
			}
			function, ok := sourcePackage.TypesInfo().Defs[functionDeclaration.Name].(*types.Func)
			if !ok || function.Origin() != function {
				return nil, &Error{
					Operation: "derive callable", Subject: functionDeclaration.Name.Name,
					Reason: "declaration has no canonical function object",
				}
			}
			contract, err := environmentcontract.Describe(function)
			if err != nil {
				return nil, &Error{
					Operation: "derive callable", Subject: function.FullName(),
					Reason: err.Error(),
				}
			}
			bodyDigest, err := callableimplementationcontract.SourceBodyDigest(
				sourcePackage.FileSet(),
				functionDeclaration.Body,
			)
			if err != nil {
				return nil, &Error{
					Operation: "derive callable", Subject: contract.Identity(),
					Reason: err.Error(),
				}
			}
			if _, duplicate := result[contract.Identity()]; duplicate {
				return nil, &Error{
					Operation: "derive callable", Subject: contract.Identity(),
					Reason: "source declaration identity is duplicated",
				}
			}
			result[contract.Identity()] = sourceCallable{
				function: function, signature: contract.Signature(), bodyDigest: bodyDigest,
			}
		}
	}
	return result, nil
}

func sameBuildProfile(left load.BuildProfile, right load.BuildProfile) bool {
	if !left.Valid() || !right.Valid() ||
		left.ToolchainVersion() != right.ToolchainVersion() ||
		left.GOOS() != right.GOOS() || left.GOARCH() != right.GOARCH() ||
		left.CgoEnabled() != right.CgoEnabled() {
		return false
	}
	leftTags := left.Tags()
	rightTags := right.Tags()
	if len(leftTags) != len(rightTags) {
		return false
	}
	for index := range leftTags {
		if leftTags[index] != rightTags[index] {
			return false
		}
	}
	return true
}

func slicesCloneModules(source []Module) []Module {
	result := make([]Module, len(source))
	for index, module := range source {
		module.callableClaims = append([]CallableDocument(nil), module.callableClaims...)
		module.certificationSources = append(
			[]CertificationSource(nil),
			module.certificationSources...,
		)
		result[index] = module
	}
	return result
}
