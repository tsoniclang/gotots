package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	runtimecontract "github.com/tsoniclang/gotots/internal/contracts/runtime"
	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeemission "github.com/tsoniclang/gotots/internal/emit/runtime"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func main() {
	contractPath := flag.String("contract", "", "runtime requirement contract")
	moduleDirectory := flag.String("module", ".", "GoToTS module directory")
	outputDirectory := flag.String("output", "", "runtime package output directory")
	profileName := flag.String("profile", "number", "integer representation")
	check := flag.Bool("check", false, "verify output without changing it")
	flag.Parse()
	if *contractPath == "" || *outputDirectory == "" {
		fail("contract and output are required")
	}
	profile, err := integerProfile(*profileName)
	if err != nil {
		fail(err.Error())
	}
	contract, err := os.ReadFile(*contractPath)
	if err != nil {
		fail(err.Error())
	}
	contractRequirements, err := runtimecontract.Decode(contract)
	if err != nil {
		fail(err.Error())
	}
	requirements, err := runtimeemission.ResolvePackageRequirements(
		contractRequirements,
	)
	if err != nil {
		fail(err.Error())
	}
	if !requirements.AllowsProfile(profile) {
		fail("runtime requirement contract does not admit profile " + profile.String())
	}
	assembled, err := runtimeemission.AssemblePackage(
		tsgo.NewFactory(),
		profile,
		requirements.RuntimeSymbols(),
		requirements.PrimitiveAliases(),
	)
	if err != nil {
		fail(err.Error())
	}
	if *check {
		info, statErr := os.Stat(*outputDirectory)
		if statErr != nil {
			fail(statErr.Error())
		}
		if !info.IsDir() {
			fail(*outputDirectory + " is not a directory")
		}
	} else {
		if err := os.MkdirAll(*outputDirectory, 0o755); err != nil {
			fail(err.Error())
		}
	}
	client, err := tsgo.StartClient(*moduleDirectory, *outputDirectory)
	if err != nil {
		fail(err.Error())
	}
	expected := make(map[string][]byte, len(assembled.Files())+1)
	for _, file := range assembled.Files() {
		printed, printErr := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if printErr != nil {
			client.Close()
			fail(printErr.Error())
		}
		relative, ok := strings.CutPrefix(
			file.OutputPath(),
			assembled.RootPath()+"/",
		)
		if !ok {
			client.Close()
			fail("runtime source is outside package root: " + file.OutputPath())
		}
		expected[filepath.Clean(filepath.FromSlash(relative))] = []byte(printed)
	}
	if err := client.Close(); err != nil {
		fail(err.Error())
	}
	expected["package.json"] = assembled.Manifest()
	if err := synchronize(*outputDirectory, expected, *check); err != nil {
		fail(err.Error())
	}
}

func integerProfile(value string) (api.IntegerRepresentation, error) {
	switch value {
	case api.IntegerRepresentationNumber.String():
		return api.IntegerRepresentationNumber, nil
	case api.IntegerRepresentationBigInt.String():
		return api.IntegerRepresentationBigInt, nil
	default:
		return api.IntegerRepresentationInvalid, fmt.Errorf(
			"integer profile %q is invalid",
			value,
		)
	}
}

func writeAtomic(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".gotots-runtime-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			os.Remove(temporaryPath)
		}
	}()
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Chmod(temporaryPath, 0o644); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	removeTemporary = false
	return nil
}

func synchronize(
	outputDirectory string,
	expected map[string][]byte,
	check bool,
) error {
	names := make([]string, 0, len(expected))
	for name := range expected {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		path := filepath.Join(outputDirectory, name)
		if check {
			actual, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read generated runtime %s: %w", name, err)
			}
			if !bytes.Equal(actual, expected[name]) {
				return fmt.Errorf("generated runtime %s is stale", name)
			}
			continue
		}
		if err := writeAtomic(path, expected[name]); err != nil {
			return err
		}
	}
	var stale []string
	err := filepath.WalkDir(
		outputDirectory,
		func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".ts" ||
				strings.HasSuffix(entry.Name(), ".d.ts") {
				return nil
			}
			relative, err := filepath.Rel(outputDirectory, path)
			if err != nil {
				return err
			}
			if _, ok := expected[relative]; !ok {
				stale = append(stale, relative)
			}
			return nil
		},
	)
	if err != nil {
		return err
	}
	sort.Strings(stale)
	if check && len(stale) != 0 {
		return fmt.Errorf("generated runtime has stale TypeScript file %s", stale[0])
	}
	for _, relative := range stale {
		if err := os.Remove(filepath.Join(outputDirectory, relative)); err != nil {
			return err
		}
	}
	return nil
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
