package load

import (
	"context"
	"errors"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"slices"

	"golang.org/x/tools/go/packages"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/toolchain"
)

type PackageRequest struct {
	Context   context.Context
	Directory string
	FileSet   *token.FileSet
	Mode      packages.LoadMode
	Overlay   map[string][]byte
	Tests     bool
}

func GoPackages(
	selectedGo toolchain.Go,
	profile environmentcontract.BuildProfile,
	request PackageRequest,
	patterns ...string,
) ([]*packages.Package, error) {
	if request.Context == nil {
		return nil, fmt.Errorf("load exact Go packages: context is nil")
	}
	if err := selectedGo.ValidateProfile(profile); err != nil {
		return nil, fmt.Errorf("load exact Go packages: %w", err)
	}
	if request.Directory == "" || len(patterns) == 0 {
		return nil, fmt.Errorf("load exact Go packages: directory or patterns are absent")
	}
	if err := selectedGo.Verify(); err != nil {
		return nil, err
	}
	evidencePath, err := reserveDriverEvidence(selectedGo.CacheRoot())
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.Remove(evidencePath) }()
	driverPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("load exact Go packages: resolve driver: %w", err)
	}
	driverPath, err = filepath.Abs(driverPath)
	if err != nil {
		return nil, fmt.Errorf("load exact Go packages: resolve driver: %w", err)
	}
	exactEnvironment := selectedGo.Environment(profile)
	if len(exactEnvironment) == 0 {
		return nil, fmt.Errorf("load exact Go packages: selected environment is absent")
	}
	environment := append(
		exactEnvironment,
		"GOPACKAGESDRIVER="+driverPath,
		toolchain.PackagesDriverModeEnvironment+"=v1",
		toolchain.PackagesDriverVersionEnvironment+"="+selectedGo.Version(),
		toolchain.PackagesDriverArchEnvironment+"="+profile.GOARCH(),
		toolchain.PackagesDriverEvidenceEnvironment+"="+evidencePath,
	)
	loaded, loadErr := packages.Load(&packages.Config{
		Context:    request.Context,
		Dir:        request.Directory,
		Fset:       request.FileSet,
		Env:        environment,
		BuildFlags: profile.BuildFlags(),
		Mode:       request.Mode,
		Overlay:    cloneOverlay(request.Overlay),
		Tests:      request.Tests,
	}, patterns...)
	if loadErr == nil {
		loadErr = joinDriverEvidence(evidencePath, loaded)
	}
	verifyErr := selectedGo.Verify()
	if loadErr != nil || verifyErr != nil {
		return nil, errors.Join(loadErr, verifyErr)
	}
	return loaded, nil
}

func cloneOverlay(source map[string][]byte) map[string][]byte {
	if source == nil {
		return nil
	}
	result := make(map[string][]byte, len(source))
	for path, content := range source {
		result[path] = slices.Clone(content)
	}
	return result
}
