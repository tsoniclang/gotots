package compositeliteral

import "github.com/tsoniclang/gotots/internal/emit/api"

func anonymousStructArtifact(
	reference api.NameReference,
) (*api.GeneratedArtifact, error) {
	var selected *api.GeneratedArtifact
	for _, request := range reference.Requests() {
		requirement, ok := request.DeclarationRequirement()
		if !ok {
			continue
		}
		artifact, demand, ok := requirement.AnonymousStruct()
		if !ok || demand != api.AnonymousStructDemandDefinition {
			continue
		}
		if selected != nil {
			return nil, &api.InvariantError{
				Reason: "anonymous struct reference has duplicate definition owners",
			}
		}
		selected = artifact
	}
	if selected == nil {
		return nil, &api.InvariantError{
			Reason: "anonymous struct reference has no definition owner",
		}
	}
	return selected, nil
}
