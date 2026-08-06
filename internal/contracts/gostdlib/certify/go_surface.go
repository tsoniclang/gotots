package certify

import (
	"bufio"
	"fmt"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
)

type goSurface struct {
	packages map[string]*goPackageSurface
	objects  map[string]goObject
}

type goPackageSurface struct {
	selected      *packages.Package
	objectsByName map[string]goObject
	methodsByType map[string][]goObject
}

type goObject struct {
	object   types.Object
	contract environmentcontract.ObjectContract
	location string
}

func loadGoSurface(
	config resolvedConfig,
	selectedToolchain toolchain,
	paths []string,
) (goSurface, error) {
	standard, err := standardPackages(config, selectedToolchain)
	if err != nil {
		return goSurface{}, err
	}
	ordered := append([]string(nil), paths...)
	sort.Strings(ordered)
	ordered = compactStrings(ordered)
	for _, packagePath := range ordered {
		if _, ok := standard[packagePath]; !ok {
			return goSurface{}, certifyError(
				"load Go surface",
				packagePath,
				"package is not in the selected standard-library set",
			)
		}
	}
	shim, err := exactGoPath(config)
	if err != nil {
		return goSurface{}, err
	}
	loaded, err := packages.Load(&packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesSizes,
		Dir:        config.repositoryRoot,
		Env:        exactGoEnvironment(selectedToolchain, shim),
		BuildFlags: selectedToolchain.profile.BuildFlags(),
	}, ordered...)
	if err != nil {
		return goSurface{}, certifyError("load Go surface", strings.Join(ordered, ","), err.Error())
	}
	if count := packages.PrintErrors(loaded); count != 0 {
		return goSurface{}, certifyError(
			"load Go surface",
			strings.Join(ordered, ","),
			fmt.Sprintf("selected toolchain reported %d package errors", count),
		)
	}
	result := goSurface{
		packages: make(map[string]*goPackageSurface, len(loaded)),
		objects:  make(map[string]goObject),
	}
	for _, selected := range loaded {
		if selected.Types == nil || selected.Fset == nil || selected.PkgPath == "" {
			return goSurface{}, certifyError(
				"load Go surface",
				selected.PkgPath,
				"package type evidence is incomplete",
			)
		}
		if _, duplicate := result.packages[selected.PkgPath]; duplicate {
			return goSurface{}, certifyError(
				"load Go surface",
				selected.PkgPath,
				"package is duplicated",
			)
		}
		current := &goPackageSurface{
			selected:      selected,
			objectsByName: make(map[string]goObject),
			methodsByType: make(map[string][]goObject),
		}
		result.packages[selected.PkgPath] = current
		names := selected.Types.Scope().Names()
		sort.Strings(names)
		for _, name := range names {
			object := selected.Types.Scope().Lookup(name)
			if object == nil {
				continue
			}
			if _, builtin := object.(*types.Builtin); !builtin &&
				!object.Pos().IsValid() {
				continue
			}
			evidence, selectedSource, err := describeGoObject(
				selectedToolchain,
				selected,
				object,
			)
			if err != nil {
				return goSurface{}, err
			}
			if !selectedSource {
				continue
			}
			if err := result.addObject(evidence); err != nil {
				return goSurface{}, err
			}
			if object.Exported() {
				if _, duplicate := current.objectsByName[name]; duplicate {
					return goSurface{}, certifyError(
						"index Go surface",
						name,
						"object name is duplicated",
					)
				}
				current.objectsByName[name] = evidence
			}
			typeName, ok := object.(*types.TypeName)
			if !ok || typeName.IsAlias() {
				continue
			}
			named, ok := types.Unalias(typeName.Type()).(*types.Named)
			if !ok {
				continue
			}
			for index := range named.NumMethods() {
				method := named.Method(index).Origin()
				methodEvidence, selectedSource, err := describeGoObject(
					selectedToolchain,
					selected,
					method,
				)
				if err != nil {
					return goSurface{}, err
				}
				if !selectedSource {
					continue
				}
				if err := result.addObject(methodEvidence); err != nil {
					return goSurface{}, err
				}
				if object.Exported() && method.Exported() {
					current.methodsByType[name] = append(
						current.methodsByType[name],
						methodEvidence,
					)
				}
			}
		}
	}
	for _, packagePath := range ordered {
		if result.packages[packagePath] == nil {
			return goSurface{}, certifyError(
				"load Go surface",
				packagePath,
				"package is absent from loaded roots",
			)
		}
	}
	return result, nil
}

func (s *goSurface) addObject(
	evidence goObject,
) error {
	identity := evidence.contract.Identity()
	if _, duplicate := s.objects[identity]; duplicate {
		return certifyError("index Go surface", identity, "object identity is duplicated")
	}
	s.objects[identity] = evidence
	return nil
}

func describeGoObject(
	selectedToolchain toolchain,
	selected *packages.Package,
	object types.Object,
) (goObject, bool, error) {
	contract, err := environmentcontract.Describe(object)
	if err != nil {
		return goObject{}, false, err
	}
	if _, builtin := object.(*types.Builtin); builtin {
		return goObject{
			object:   object,
			contract: contract,
			location: "builtin",
		}, true, nil
	}
	if !object.Pos().IsValid() {
		return goObject{}, false, certifyError(
			"describe Go object",
			contract.Identity(),
			"source position is absent",
		)
	}
	location, selectedSource, err := selectedGoSourceLocation(
		selectedToolchain.root,
		selected.Fset,
		object.Pos(),
	)
	if err != nil {
		return goObject{}, false, certifyError(
			"describe Go object",
			contract.Identity(),
			err.Error(),
		)
	}
	if !selectedSource {
		return goObject{}, false, nil
	}
	return goObject{
		object:   object,
		contract: contract,
		location: location,
	}, true, nil
}

func selectedGoSourceLocation(
	root string,
	files *token.FileSet,
	position token.Pos,
) (string, bool, error) {
	if root == "" || files == nil || !position.IsValid() {
		return "", false, fmt.Errorf("source position is invalid")
	}
	resolved := files.PositionFor(position, false)
	if !resolved.IsValid() {
		return "", false, fmt.Errorf("source position is invalid")
	}
	sourceRoot := filepath.Join(root, "src")
	relative, err := filepath.Rel(sourceRoot, resolved.Filename)
	if err != nil {
		return "", false, err
	}
	if relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false, nil
	}
	return filepath.ToSlash(relative) + ":" +
		fmt.Sprintf("%d:%d", resolved.Line, resolved.Column), true, nil
}

func standardPackages(
	config resolvedConfig,
	selectedToolchain toolchain,
) (map[string]struct{}, error) {
	arguments := append([]string{"list"}, selectedToolchain.profile.BuildFlags()...)
	arguments = append(arguments, "std")
	command := exec.Command(config.goBinary, arguments...)
	command.Dir = config.repositoryRoot
	command.Env = exactGoEnvironment(selectedToolchain, filepath.Dir(config.goBinary))
	output, err := command.StdoutPipe()
	if err != nil {
		return nil, certifyError("list standard packages", config.goBinary, err.Error())
	}
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		return nil, certifyError("list standard packages", config.goBinary, err.Error())
	}
	result := make(map[string]struct{})
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name != "" {
			result[name] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, certifyError("list standard packages", config.goBinary, err.Error())
	}
	if err := command.Wait(); err != nil {
		return nil, commandError("list standard packages", config.goBinary, err)
	}
	if len(result) == 0 {
		return nil, certifyError("list standard packages", config.goBinary, "set is empty")
	}
	return result, nil
}

func exactGoPath(config resolvedConfig) (string, error) {
	directory, err := os.MkdirTemp(config.scratchDirectory, "go-toolchain-")
	if err != nil {
		return "", certifyError("select Go toolchain", config.scratchDirectory, err.Error())
	}
	shim := filepath.Join(directory, "go")
	if err := os.Symlink(config.goBinary, shim); err != nil {
		return "", certifyError("select Go toolchain", shim, err.Error())
	}
	return directory, nil
}

func exactGoEnvironment(selectedToolchain toolchain, binaryDirectory string) []string {
	selected := selectedToolchain.profile.Environment(os.Environ())
	result := make([]string, 0, len(selected)+3)
	for _, value := range selected {
		if strings.HasPrefix(value, "PATH=") ||
			strings.HasPrefix(value, "GOROOT=") ||
			strings.HasPrefix(value, "GOTOOLCHAIN=") {
			continue
		}
		result = append(result, value)
	}
	result = append(
		result,
		"PATH="+binaryDirectory+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GOROOT="+selectedToolchain.root,
		"GOTOOLCHAIN=local",
	)
	return result
}
