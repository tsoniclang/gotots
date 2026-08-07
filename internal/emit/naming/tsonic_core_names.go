package naming

import (
	"strconv"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (n *File) TsonicCore(
	symbol tsoniccore.Symbol,
) (api.NameReference, error) {
	declaration, err := tsoniccore.Resolve(symbol)
	if err != nil {
		return api.NameReference{}, err
	}
	phase := api.ImportPhaseValue
	if declaration.Phase() == tsoniccore.PhaseType {
		phase = api.ImportPhaseType
	}
	localName := n.tsonicCore[symbol]
	if localName == "" {
		localName = declaration.Export()
		if n.sourceNameExists(localName) || n.hasImportName(localName) {
			base := declaration.Export() + "__from_tsonic_core"
			localName = base
			for suffix := uint64(1); n.sourceNameExists(localName) ||
				n.hasImportName(localName); suffix++ {
				localName = base + "_" + strconv.FormatUint(suffix, 10)
			}
		}
		n.importNames[localName] = struct{}{}
		n.tsonicCore[symbol] = localName
	}
	request, err := api.NewImportRequest(
		n.factory,
		phase,
		declaration.Module(),
		declaration.Export(),
		localName,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewNameReference(localName, request)
}
