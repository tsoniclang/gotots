package naming

import (
	"github.com/tsoniclang/gotots/internal/contracts/goabi"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (n *File) SourceDataLayout(layout goabi.Layout) (api.NameReference, error) {
	export, err := layout.Export()
	if err != nil {
		return api.NameReference{}, err
	}
	if n.sourceLayouts == nil {
		n.sourceLayouts = make(map[goabi.Layout]string)
	}
	local := n.sourceLayouts[layout]
	if local == "" {
		local = n.allocateImportName(export, "go_abi")
		n.importNames[local] = struct{}{}
		n.sourceLayouts[layout] = local
	}
	request, err := api.NewImportRequest(n.factory, api.ImportPhaseValue, goabi.Module, export, local)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewNameReference(local, request)
}
