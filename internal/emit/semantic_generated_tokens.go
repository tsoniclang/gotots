package emit

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/generic/semanticname"
)

func (s *programSession) semanticPackageToken(
	sourcePackage *types.Package,
) (string, error) {
	if s == nil || s.registry == nil || sourcePackage == nil {
		return "", &ScheduleError{Reason: "semantic package token owner is invalid"}
	}
	qualifier := s.registry.ImportQualifier(sourcePackage)
	if qualifier == "" {
		return "", &ScheduleError{
			Object: sourcePackage.Path(),
			Reason: "semantic package token is absent",
		}
	}
	return qualifier, nil
}

func (s *programSession) semanticNamedTypeToken(
	object *types.TypeName,
) (string, error) {
	if object == nil {
		return "", &ScheduleError{Reason: "semantic named type is nil"}
	}
	if object.Pkg() == nil {
		return semanticname.Identifier(object.Name()), nil
	}
	if object.Parent() != object.Pkg().Scope() {
		return "", &ScheduleError{
			Object: object.Name(),
			Reason: "semantic operation local type has no lexical owner",
		}
	}
	qualifier, err := s.semanticPackageToken(object.Pkg())
	if err != nil {
		return "", err
	}
	return qualifier + "$" + semanticname.Identifier(object.Name()), nil
}
