package naming

import (
	"strings"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/output"
)

type generatedInterfaceContractImports struct {
	typeName     string
	contractName string
	guardName    string
	requests     []api.RootRequest
}

func (n *File) generatedArtifactLocalName(
	artifact *api.GeneratedArtifact,
	exported string,
) string {
	identity := generatedArtifactImport{
		artifact: artifact,
		exported: exported,
	}
	if selected := n.artifactImports[identity]; selected != "" {
		return selected
	}
	selected := generatedArtifactPreferredLocalName(artifact, exported)
	if n.lexicalNameExists(selected) {
		if selected != exported && !n.lexicalNameExists(exported) {
			selected = exported
		} else {
			selected = n.allocateImportName(
				selected,
				generatedArtifactImportQualifier(artifact),
			)
		}
	} else {
		n.importNames[selected] = struct{}{}
	}
	n.artifactImports[identity] = selected
	return selected
}

func generatedArtifactPreferredLocalName(
	artifact *api.GeneratedArtifact,
	exported string,
) string {
	if artifact == nil || exported == "" {
		return exported
	}
	base := artifact.TargetName()
	if !strings.HasPrefix(exported, base) {
		return exported
	}
	suffix := strings.TrimPrefix(exported, base)
	preferred := ""
	switch artifact.Kind() {
	case api.GeneratedArtifactMapSpecialization:
		preferred = "GoMap"
	case api.GeneratedArtifactInterfaceAdapter:
		preferred = "GoInterfaceAdapter"
	case api.GeneratedArtifactAnonymousInterface:
		preferred = "GoInterface"
	case api.GeneratedArtifactProviderInterfaceBridge:
		preferred = "GoProviderInterfaceBridge"
		if strings.HasPrefix(base, "$goProviderProfileBridge$") {
			preferred = "GoProviderProfileBridge"
		}
	case api.GeneratedArtifactProviderStatefulRepresentation:
		preferred = "GoProviderState"
	case api.GeneratedArtifactDeferredCallableRegistry:
		preferred = "DeferredCallableRegistry"
	default:
		return exported
	}
	return preferred + suffix
}

func (n *File) interfaceContractImports(
	artifact *api.GeneratedArtifact,
	baseName string,
) (generatedInterfaceContractImports, error) {
	modulePath, err := output.ModuleSpecifier(
		n.targetPath,
		artifact.OutputPath(),
	)
	if err != nil {
		return generatedInterfaceContractImports{}, err
	}
	exports := []struct {
		name  string
		phase api.ImportPhase
	}{
		{baseName, api.ImportPhaseType},
		{interfaceContractName(baseName), api.ImportPhaseValue},
		{interfaceGuardName(baseName), api.ImportPhaseValue},
	}
	result := generatedInterfaceContractImports{
		requests: make([]api.RootRequest, 0, len(exports)),
	}
	localNames := []*string{
		&result.typeName,
		&result.contractName,
		&result.guardName,
	}
	for index, exported := range exports {
		localName := n.generatedArtifactLocalName(artifact, exported.name)
		request, requestErr := api.NewImportRequest(
			n.factory,
			exported.phase,
			modulePath,
			exported.name,
			localName,
		)
		if requestErr != nil {
			return generatedInterfaceContractImports{}, requestErr
		}
		*localNames[index] = localName
		result.requests = append(result.requests, request)
	}
	return result, nil
}
