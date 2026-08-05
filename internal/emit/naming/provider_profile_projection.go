package naming

import (
	"go/types"
	"slices"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

type providerProfileProjection struct {
	certificates map[string]gostdlib.ProviderCallableProfileInterface
	selected     map[string]gostdlib.ProviderCallableProfileInterface
	visited      map[types.Type]struct{}
}

func providerProfileBridgeClosure(
	source *types.Named,
	profile []gostdlib.ProviderCallableProfileInterface,
) ([]gostdlib.ProviderCallableProfileInterface, error) {
	if source == nil || source.Obj() == nil || len(profile) == 0 {
		return nil, &api.NameError{
			Reason: "provider-profile bridge projection input is invalid",
		}
	}
	projection := providerProfileProjection{
		certificates: make(
			map[string]gostdlib.ProviderCallableProfileInterface,
			len(profile),
		),
		selected: make(map[string]gostdlib.ProviderCallableProfileInterface),
		visited:  make(map[types.Type]struct{}),
	}
	for _, certificate := range profile {
		identity := certificate.SourceIdentity()
		if !certificate.Valid() ||
			certificate.ProviderInterface().Mode() !=
				gostdlib.ProviderInterfaceModeBridge {
			return nil, &api.NameError{
				Name:   identity,
				Reason: "provider-profile bridge certificate is invalid",
			}
		}
		if _, duplicate := projection.certificates[identity]; duplicate {
			return nil, &api.NameError{
				Name:   identity,
				Reason: "provider-profile bridge certificate is duplicated",
			}
		}
		projection.certificates[identity] = certificate
	}
	sourceIdentity, err := gostdlibsource.ObjectIdentity(source.Origin().Obj())
	if err != nil {
		return nil, err
	}
	if _, certified := projection.certificates[sourceIdentity]; !certified {
		return nil, &api.NameError{
			Name:   sourceIdentity,
			Reason: "provider-profile bridge source certificate is absent",
		}
	}
	if err := projection.collect(source); err != nil {
		return nil, err
	}
	selected := make(
		[]gostdlib.ProviderCallableProfileInterface,
		0,
		len(projection.selected),
	)
	for _, certificate := range projection.selected {
		selected = append(selected, certificate)
	}
	slices.SortFunc(selected, func(
		left gostdlib.ProviderCallableProfileInterface,
		right gostdlib.ProviderCallableProfileInterface,
	) int {
		if compared := strings.Compare(
			left.SourceIdentity(),
			right.SourceIdentity(),
		); compared != 0 {
			return compared
		}
		return strings.Compare(
			left.TargetFingerprint(),
			right.TargetFingerprint(),
		)
	})
	return selected, nil
}

func (p *providerProfileProjection) collect(source types.Type) error {
	if source == nil {
		return nil
	}
	source = types.Unalias(source)
	if _, seen := p.visited[source]; seen {
		return nil
	}
	p.visited[source] = struct{}{}
	switch selected := source.(type) {
	case *types.Named:
		if contract, ok := selected.Underlying().(*types.Interface); ok {
			identity, err := gostdlibsource.ObjectIdentity(
				selected.Origin().Obj(),
			)
			if err != nil {
				return err
			}
			certificate, profiled := p.certificates[identity]
			if !profiled {
				return nil
			}
			p.selected[identity] = certificate
			return p.collectInterface(contract.Complete())
		}
		return p.collect(selected.Underlying())
	case *types.Interface:
		return p.collectInterface(selected.Complete())
	case *types.Struct:
		for index := range selected.NumFields() {
			if err := p.collect(selected.Field(index).Type()); err != nil {
				return err
			}
		}
	case *types.Pointer:
		return p.collect(selected.Elem())
	case *types.Slice:
		return p.collect(selected.Elem())
	case *types.Array:
		return p.collect(selected.Elem())
	case *types.Map:
		if err := p.collect(selected.Key()); err != nil {
			return err
		}
		return p.collect(selected.Elem())
	case *types.Chan:
		return p.collect(selected.Elem())
	case *types.Signature:
		return p.collectSignature(selected)
	case *types.Tuple:
		for index := range selected.Len() {
			if err := p.collect(selected.At(index).Type()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *providerProfileProjection) collectInterface(
	contract *types.Interface,
) error {
	if contract == nil {
		return nil
	}
	for index := range contract.NumMethods() {
		signature, ok := contract.Method(index).Type().(*types.Signature)
		if !ok {
			return &api.NameError{
				Name:   contract.Method(index).Name(),
				Reason: "provider-profile interface method signature is invalid",
			}
		}
		if err := p.collectSignature(signature); err != nil {
			return err
		}
	}
	return nil
}

func (p *providerProfileProjection) collectSignature(
	signature *types.Signature,
) error {
	if signature == nil {
		return nil
	}
	if err := p.collect(signature.Params()); err != nil {
		return err
	}
	return p.collect(signature.Results())
}
