package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
)

func main() {
	repository := flag.String("repository", "", "GoToTS repository root")
	provider := flag.String("provider", "", "gostdlib package root")
	manifest := flag.String("manifest", "", "generated provider manifest")
	modules := flag.String("modules", "", "authoritative provider module map")
	facets := flag.String("facets", "", "authoritative compiler-facet map")
	runtimeContract := flag.String("runtime", "", "generated runtime contract")
	tsconfig := flag.String("tsconfig", "", "provider TypeScript project")
	scratch := flag.String("scratch", "", "bounded certification scratch directory")
	goBinary := flag.String("go", "go", "selected Go toolchain binary")
	goos := flag.String("goos", runtime.GOOS, "selected source GOOS")
	goarch := flag.String("goarch", runtime.GOARCH, "selected source GOARCH")
	cgo := flag.Bool("cgo", false, "select cgo-enabled source")
	tags := flag.String("tags", "", "comma-separated selected build tags")
	backend := flag.String("backend", "", "provider backend")
	minimumGo := flag.String("minimum-go", "", "minimum selected Go version")
	maximumGo := flag.String("maximum-go", "", "maximum selected Go version")
	check := flag.Bool("check", false, "verify the checked manifest without writing")
	flag.Parse()
	var buildTags []string
	if *tags != "" {
		buildTags = strings.Split(*tags, ",")
	}
	buildProfile, err := environmentcontract.NewBuildProfile(
		*goos,
		*goarch,
		*cgo,
		buildTags,
	)
	if err != nil {
		fail(err)
	}
	config := certify.Config{
		RepositoryRoot:      *repository,
		ProviderRoot:        *provider,
		ManifestPath:        *manifest,
		ModuleMapPath:       *modules,
		FacetMapPath:        *facets,
		RuntimeContractPath: *runtimeContract,
		TSConfigPath:        *tsconfig,
		ScratchDirectory:    *scratch,
		GoBinary:            *goBinary,
		BuildProfile:        buildProfile,
		Backend:             *backend,
		MinimumGoVersion:    *minimumGo,
		MaximumGoVersion:    *maximumGo,
	}
	generated, err := certify.Generate(config)
	if err != nil {
		fail(err)
	}
	if *check {
		current, err := os.ReadFile(*manifest)
		if err != nil {
			fail(err)
		}
		if !bytes.Equal(current, generated) {
			fail(fmt.Errorf("gostdlib manifest is stale"))
		}
		return
	}
	if err := writeAtomic(*manifest, generated); err != nil {
		fail(err)
	}
}

func writeAtomic(path string, content []byte) error {
	if path == "" {
		return fmt.Errorf("manifest path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".gostdlib-manifest-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
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
	return os.Rename(temporaryPath, path)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
