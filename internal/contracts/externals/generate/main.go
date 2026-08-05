package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	externalcertify "github.com/tsoniclang/gotots/internal/contracts/externals/certify"
)

func main() {
	var repository string
	var provider string
	var manifest string
	var bindings string
	var tsconfig string
	var standardLibraryManifest string
	var standardLibraryRuntime string
	var goos string
	var goarch string
	var tags string
	var backend string
	var check bool
	flag.StringVar(&repository, "repository", "", "repository root")
	flag.StringVar(&provider, "provider", "", "external provider root")
	flag.StringVar(&manifest, "manifest", "", "checked manifest path")
	flag.StringVar(&bindings, "bindings", "", "binding map path")
	flag.StringVar(&tsconfig, "tsconfig", "", "provider tsconfig path")
	flag.StringVar(
		&standardLibraryManifest,
		"gostdlib-manifest",
		"",
		"checked gostdlib manifest path",
	)
	flag.StringVar(
		&standardLibraryRuntime,
		"gostdlib-runtime",
		"",
		"checked gostdlib runtime-contract path",
	)
	flag.StringVar(&goos, "goos", "", "selected GOOS")
	flag.StringVar(&goarch, "goarch", "", "selected GOARCH")
	flag.StringVar(&tags, "tags", "", "comma-separated build tags")
	flag.StringVar(&backend, "backend", "", "provider backend")
	flag.BoolVar(&check, "check", false, "verify without writing")
	flag.Parse()
	selectedTags := []string{}
	if tags != "" {
		selectedTags = strings.Split(tags, ",")
	}
	profile, err := environmentcontract.NewBuildProfileForToolchain(
		runtime.Version(),
		goos,
		goarch,
		false,
		selectedTags,
	)
	if err != nil {
		fatalf("build profile: %v", err)
	}
	config := externalcertify.Config{
		RepositoryRoot:              repository,
		ProviderRoot:                provider,
		ManifestPath:                manifest,
		BindingMapPath:              bindings,
		TSConfigPath:                tsconfig,
		StandardLibraryManifestPath: standardLibraryManifest,
		StandardLibraryRuntimePath:  standardLibraryRuntime,
		BuildProfile:                profile,
		Backend:                     backend,
	}
	generated, err := externalcertify.Generate(config)
	if err != nil {
		fatalf("generate: %v", err)
	}
	if check {
		checked, err := os.ReadFile(manifest)
		if err != nil {
			fatalf("read checked manifest: %v", err)
		}
		if !bytes.Equal(checked, generated) {
			fatalf("checked manifest differs from generated evidence")
		}
		return
	}
	if err := os.WriteFile(manifest, generated, 0o644); err != nil {
		fatalf("write manifest: %v", err)
	}
}

func fatalf(format string, values ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", values...)
	os.Exit(1)
}
