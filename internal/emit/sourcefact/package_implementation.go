package sourcefact

import (
	"github.com/tsoniclang/gotots/internal/contracts/sourceimplementation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const packageImplementationSchema = "gotots-go-package-implementation-fact-v1"

func PackageImplementationArguments(
	factory tsgo.Factory,
	implementation sourceimplementation.Implementation,
	outputPath string,
) ([]tsgo.Expression, error) {
	envelope := implementation.EquivalenceEnvelope()
	if outputPath == "" || !envelope.Valid() {
		return nil, &Error{Reason: "package implementation fact is incomplete"}
	}
	arguments := []tsgo.Expression{
		text(factory, packageImplementationSchema),
		text(factory, "package-replacement"),
		text(factory, implementation.PackagePath()),
		text(factory, implementation.ModulePath()),
		text(factory, implementation.ModuleVersion()),
		text(factory, outputPath),
		text(factory, implementation.Digest()),
		text(factory, implementation.SourceDigest()),
		text(factory, string(envelope.Kind)),
		text(factory, envelope.RelaxedBehavior),
		count(factory, len(envelope.PreservedObservables)),
	}
	for index, observable := range envelope.PreservedObservables {
		arguments = append(arguments, count(factory, index), text(factory, observable))
	}
	arguments = append(arguments, count(factory, len(envelope.Evidence)))
	for index, evidence := range envelope.Evidence {
		arguments = append(arguments, count(factory, index), text(factory, evidence))
	}
	exports := implementation.Exports()
	arguments = append(arguments, count(factory, len(exports)))
	for index, exported := range exports {
		arguments = append(
			arguments,
			count(factory, index),
			text(factory, exported.Name()),
			text(factory, exported.TypeString()),
			text(factory, exported.Fingerprint()),
		)
	}
	modules := implementation.PrivateModules()
	arguments = append(arguments, count(factory, len(modules)))
	for index, module := range modules {
		arguments = append(
			arguments,
			count(factory, index),
			text(factory, module.GoFile()),
			text(factory, module.SourceDigest()),
			count(factory, len(module.Exports())),
		)
		for exportIndex, exported := range module.Exports() {
			arguments = append(
				arguments,
				count(factory, exportIndex),
				text(factory, exported.Name()),
				text(factory, exported.TypeString()),
				text(factory, exported.Fingerprint()),
			)
		}
	}
	return arguments, nil
}
