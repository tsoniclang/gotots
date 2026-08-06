package gostdlib_test

import "github.com/tsoniclang/gotots/internal/contracts/gostdlib"

func validDocument() gostdlib.Document {
	return gostdlib.Document{
		SchemaVersion:    gostdlib.SchemaVersion,
		PackageName:      gostdlib.PackageName,
		PackageVersion:   "0.0.0",
		Backend:          "node",
		GoVersion:        "go1.26.4",
		MinimumGoVersion: "go1.26.4",
		MaximumGoVersion: "go1.26.4",
		GOOS:             "linux",
		GOARCH:           "amd64",
		BuildTags:        []string{},
		RuntimeDigest:    digest('a'),
		ProviderDigest:   digest('c'),
		Modules: []gostdlib.ModuleDocument{{
			GoImportPath: "strings",
			Specifier:    "@gotots/gostdlib/strings.js",
			SourcePath:   "src/strings.ts",
			Bindings: []gostdlib.BindingDocument{{
				Identity:            "strings|kind=4|receiver=|name=Contains",
				Kind:                gostdlib.BindingFunction,
				Access:              gostdlib.AccessExport,
				Effect:              gostdlib.EffectSynchronous,
				Export:              "Contains",
				SourceSignature:     "func(s, substr string) bool|params=s,substr|results=",
				SourceLocation:      "strings/strings.go:1:1",
				ImplementationOwner: "src/strings.ts",
				TargetFingerprint:   digest('b'),
			}},
		}},
	}
}

func digest(character byte) string {
	value := make([]byte, 64)
	for index := range value {
		value[index] = character
	}
	return string(value)
}
