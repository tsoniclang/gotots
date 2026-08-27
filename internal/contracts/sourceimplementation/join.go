package sourceimplementation

import (
	"path/filepath"

	"github.com/tsoniclang/gotots/internal/load"
)

func (p *Prepared) Join(program *load.Program) (*Certificate, error) {
	if p == nil || program == nil || !p.buildProfile.Valid() ||
		!p.certificate.Valid() {
		return nil, &Error{
			Operation: "join",
			Reason:    "prepared evidence or selected program is invalid",
		}
	}
	if !sameBuildProfile(p.buildProfile, program.BuildProfile()) {
		return nil, &Error{
			Operation: "join build profile",
			Reason:    "selected profile differs",
		}
	}
	for _, implementation := range p.certificate.Implementations() {
		selected := program.PackageByPath(implementation.packagePath)
		if selected == nil || selected.Kind() != load.PackageSource ||
			selected.ModulePath() != implementation.modulePath ||
			selected.ModuleVersion() != implementation.moduleVersion {
			return nil, &Error{
				Operation: "join package",
				Subject:   implementation.packagePath,
				Reason:    "selected source package identity differs",
			}
		}
		files := make(map[string]struct{}, len(selected.Files()))
		for _, sourceFile := range selected.Files() {
			files[filepath.Base(sourceFile.Path())] = struct{}{}
		}
		for _, module := range implementation.privateModules {
			if _, exists := files[module.goFile]; !exists {
				return nil, &Error{
					Operation: "join private module",
					Subject:   module.goFile,
					Reason:    "selected Go source file is absent",
				}
			}
		}
	}
	certificate := p.certificate
	certificate.byPath = make(
		map[string]Implementation,
		len(p.certificate.byPath),
	)
	for path, implementation := range p.certificate.byPath {
		certificate.byPath[path] = implementation
	}
	return &certificate, nil
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
