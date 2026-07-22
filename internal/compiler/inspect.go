// Package compiler sequences the compilation phases. It holds no semantic
// cases: it orders the phase owners, runs each stage's blocking independent
// verifier before any downstream stage consumes the artifact, and surfaces
// typed results. cmd/gotots is its only caller; there is exactly one
// compilation route — inspection uses the same request and loader as
// generation will.
package compiler

import (
	"github.com/tsoniclang/gotots/internal/language/analyze"
	"github.com/tsoniclang/gotots/internal/source"
	"github.com/tsoniclang/gotots/internal/stagecheck"
)

// InspectConstructs resolves a compilation request into a verified
// whole-workspace construct inventory:
//
//	request -> SourceUniverse -> [verify] -> SyntaxInventory -> [verify] -> report
//
// A failed stage verifier blocks every downstream stage; there is no partial
// or unverified artifact.
func InspectConstructs(req source.Request) (*analyze.WorkspaceInventory, error) {
	ws, err := source.LoadWorkspace(req)
	if err != nil {
		return nil, err
	}
	if err := stagecheck.VerifySourceUniverse(ws, req); err != nil {
		return nil, err
	}
	inventory, err := analyze.BuildWorkspaceInventory(ws)
	if err != nil {
		return nil, err
	}
	if err := stagecheck.VerifySyntaxInventory(ws, inventory); err != nil {
		return nil, err
	}
	return inventory, nil
}
