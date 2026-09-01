package sourcefact

import (
	"go/types"
	"sort"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	attribute "github.com/tsoniclang/gotots/internal/emit/marker/attribute"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const packageInitializationSchema = "gotots-go-package-initialization-fact-v2"

type PackageInitializer struct {
	Targets []*types.Var
}

func SourcePackageInitialization(
	context api.Context,
	targetName string,
	sourcePackage *load.Package,
	outputPath string,
	sourceDigest string,
	storageCount int,
	initializerTargets [][]*types.Var,
	initFunctions []*types.Func,
) (api.StatementEmission, error) {
	if sourcePackage == nil {
		return api.StatementEmission{}, &Error{
			Reason: "package initialization source owner is absent",
		}
	}
	initializers := make([]PackageInitializer, len(initializerTargets))
	for index := range initializerTargets {
		initializers[index] = PackageInitializer{Targets: initializerTargets[index]}
	}
	imports := make([]string, 0, len(sourcePackage.Types().Imports()))
	for _, imported := range sourcePackage.Types().Imports() {
		imports = append(imports, imported.Path())
	}
	sort.Strings(imports)
	ownerKind, contractKey := PackageOwner(sourcePackage)
	return PackageInitialization(
		context,
		targetName,
		sourcePackage.Path(),
		sourcePackage.ModulePath(),
		sourcePackage.ModuleVersion(),
		ownerKind,
		contractKey,
		outputPath,
		sourceDigest,
		imports,
		storageCount,
		initializers,
		initFunctions,
	)
}

func PackageInitialization(
	context api.Context,
	targetName string,
	packagePath string,
	modulePath string,
	moduleVersion string,
	ownerKind string,
	contractKey string,
	outputPath string,
	sourceDigest string,
	imports []string,
	storageCount int,
	initializers []PackageInitializer,
	initFunctions []*types.Func,
) (api.StatementEmission, error) {
	if targetName == "" || packagePath == "" ||
		outputPath == "" || sourceDigest == "" || storageCount < 0 ||
		!validAuthoredOwner(ownerKind, modulePath, moduleVersion, contractKey) {
		return api.StatementEmission{}, &Error{Reason: "package initialization fact is incomplete"}
	}
	arguments := []tsgo.Expression{
		text(context.Factory(), packageInitializationSchema),
		text(context.Factory(), packagePath),
		text(context.Factory(), modulePath),
		text(context.Factory(), moduleVersion),
		text(context.Factory(), ownerKind),
		text(context.Factory(), contractKey),
		text(context.Factory(), outputPath),
		text(context.Factory(), sourceDigest),
		count(context.Factory(), len(imports)),
	}
	for index, imported := range imports {
		if imported == "" {
			return api.StatementEmission{}, &Error{Subject: packagePath, Reason: "package import identity is empty"}
		}
		arguments = append(
			arguments,
			count(context.Factory(), index),
			text(context.Factory(), imported),
		)
	}
	arguments = append(arguments,
		count(context.Factory(), storageCount),
		count(context.Factory(), len(initializers)),
	)
	for index, initializer := range initializers {
		if len(initializer.Targets) == 0 {
			return api.StatementEmission{}, &Error{Subject: packagePath, Reason: "package initializer has no targets"}
		}
		arguments = append(
			arguments,
			count(context.Factory(), index),
			count(context.Factory(), len(initializer.Targets)),
		)
		for targetIndex, target := range initializer.Targets {
			if target == nil {
				return api.StatementEmission{}, &Error{Subject: packagePath, Reason: "package initializer target is nil"}
			}
			identity := "blank"
			if target.Name() != "_" {
				contract, err := environmentcontract.Describe(target)
				if err != nil {
					return api.StatementEmission{}, err
				}
				identity = contract.Identity()
			}
			arguments = append(
				arguments,
				count(context.Factory(), targetIndex),
				text(context.Factory(), identity),
			)
		}
	}
	arguments = append(arguments, count(context.Factory(), len(initFunctions)))
	for index, function := range initFunctions {
		if function == nil {
			return api.StatementEmission{}, &Error{Subject: packagePath, Reason: "package init function is nil"}
		}
		contract, err := environmentcontract.Describe(function)
		if err != nil {
			return api.StatementEmission{}, err
		}
		arguments = append(
			arguments,
			count(context.Factory(), index),
			text(context.Factory(), contract.Identity()),
		)
	}
	return attribute.Apply(
		context,
		context.Factory().TypeQueryNode(context.Factory().Identifier(targetName), nil),
		api.RuntimeSourceOperationFact,
		arguments...,
	)
}
