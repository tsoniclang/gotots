package api

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func NewSideEffectImportRequest(
	factory tsgo.Factory,
	modulePath string,
) (RootRequest, error) {
	if modulePath == "" {
		return RootRequest{}, &RootRequestError{Reason: "module path is empty"}
	}
	return RootRequest{payload: &rootRequestPayload{
		owner: RootRequestOwner{
			kind:          RootRequestImport,
			importBinding: ImportBindingSideEffect,
			modulePath:    modulePath,
		},
		importPhase:     ImportPhaseValue,
		moduleSpecifier: factory.StringLiteral(modulePath, tsgo.TokenFlagsNone),
	}}, nil
}
