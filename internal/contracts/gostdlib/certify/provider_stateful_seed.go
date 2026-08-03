package certify

import "github.com/tsoniclang/gotots/internal/contracts/gostdlib"

type providerStatefulProfileSeed struct {
	SourceIdentity string                                 `json:"sourceIdentity"`
	Specifier      string                                 `json:"specifier"`
	SourcePath     string                                 `json:"sourcePath"`
	Export         string                                 `json:"export"`
	Interfaces     []providerCallableProfileInterfaceSeed `json:"interfaces"`
	TypeArguments  []string                               `json:"typeArguments"`
	Operations     []gostdlib.FacetCapability             `json:"operations,omitempty"`
}
