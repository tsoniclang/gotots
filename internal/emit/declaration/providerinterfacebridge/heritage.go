package providerinterfacebridge

import (
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func implementedContractNames(
	base string,
	capabilities []capabilitySelection,
) []string {
	names := make([]string, 0, len(capabilities)+1)
	seen := make(map[string]struct{}, len(capabilities)+1)
	appendName := func(name string) {
		if name == "" {
			return
		}
		if _, exists := seen[name]; exists {
			return
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	appendName(base)
	for _, capability := range capabilities {
		appendName(capability.canonical.TypeName())
	}
	return names
}

func implementsHeritage(
	factory tsgo.Factory,
	names []string,
) tsgo.HeritageClause {
	implemented := make([]tsgo.ExpressionWithTypeArguments, 0, len(names))
	for _, name := range names {
		implemented = append(
			implemented,
			factory.ExpressionWithTypeArguments(
				factory.Identifier(name),
				nil,
			),
		)
	}
	return factory.HeritageClause(
		tsgo.HeritageClauseTokenKindImplementsKeyword,
		implemented,
	)
}
