package load

import (
	"context"
	"fmt"
	"strings"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/toolchain"
)

type toolchainPackageMembership struct {
	standard map[string]struct{}
	command  map[string]struct{}
}

func loadToolchainPackageMembership(
	ctx context.Context,
	selectedGo toolchain.Go,
	profile environmentcontract.BuildProfile,
	directory string,
) (toolchainPackageMembership, error) {
	standard, err := listToolchainPackageSet(
		ctx,
		selectedGo,
		profile,
		directory,
		"std",
	)
	if err != nil {
		return toolchainPackageMembership{}, err
	}
	command, err := listToolchainPackageSet(
		ctx,
		selectedGo,
		profile,
		directory,
		"cmd",
	)
	if err != nil {
		return toolchainPackageMembership{}, err
	}
	for packagePath := range command {
		if _, overlap := standard[packagePath]; overlap {
			return toolchainPackageMembership{}, fmt.Errorf(
				"selected toolchain package %q belongs to both std and cmd sets",
				packagePath,
			)
		}
	}
	return toolchainPackageMembership{
		standard: standard,
		command:  command,
	}, nil
}

func listToolchainPackageSet(
	ctx context.Context,
	selectedGo toolchain.Go,
	profile environmentcontract.BuildProfile,
	directory string,
	pattern string,
) (map[string]struct{}, error) {
	arguments := append([]string{"list"}, profile.BuildFlags()...)
	arguments = append(arguments, pattern)
	payload, err := selectedGo.Output(ctx, directory, profile, arguments...)
	if err != nil {
		return nil, fmt.Errorf("list selected toolchain %s packages: %w", pattern, err)
	}
	result := make(map[string]struct{})
	for _, packagePath := range strings.Fields(string(payload)) {
		if _, duplicate := result[packagePath]; duplicate {
			return nil, fmt.Errorf(
				"selected toolchain %s set contains duplicate package %q",
				pattern,
				packagePath,
			)
		}
		result[packagePath] = struct{}{}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("selected toolchain %s set is empty", pattern)
	}
	return result, nil
}

func (m toolchainPackageMembership) standardContains(packagePath string) bool {
	_, ok := m.standard[packagePath]
	return ok
}

func (m toolchainPackageMembership) commandContains(packagePath string) bool {
	_, ok := m.command[packagePath]
	return ok
}
