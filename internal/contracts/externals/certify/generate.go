package certify

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/types"
	"os"
	"sort"

	"github.com/tsoniclang/gotots/internal/contracts/externals"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	runtimecontract "github.com/tsoniclang/gotots/internal/contracts/runtime"
)

func Generate(config Config) ([]byte, error) {
	resolved, err := resolveConfig(config)
	if err != nil {
		return nil, err
	}
	if err := verifyProviderTypecheck(resolved); err != nil {
		return nil, err
	}
	seeds, err := readSeeds(resolved.bindingMapPath)
	if err != nil {
		return nil, err
	}
	providerPackage, err := readProviderPackage(resolved, seeds)
	if err != nil {
		return nil, err
	}
	standardLibraryDigest, providerIntegerRepresentation, err := readStandardLibraryContract(
		resolved.standardLibraryManifestPath,
		resolved.standardLibraryRuntimePath,
	)
	if err != nil {
		return nil, err
	}
	sources, err := loadSourcePackages(resolved, seeds)
	if err != nil {
		return nil, err
	}
	targets, err := inspectProviderTargets(resolved, seeds, sources)
	if err != nil {
		return nil, err
	}
	bindings := make([]externals.BindingDocument, 0, len(seeds))
	for _, seed := range seeds {
		selected, ok := sources[seed.SourcePackage].functions[seed.SourceName]
		if !ok || selected.body || !linkableSignature(selected.signature) {
			return nil, certifyError(
				"build binding",
				seed.SourcePackage+"."+seed.SourceName,
				"source is absent, body-owning, generic, or variadic",
			)
		}
		binding := externals.BindingDocument{
			SourceIdentity:      selected.contract.Identity(),
			SourceSignature:     selected.contract.Signature(),
			SourceModulePath:    selected.modulePath,
			SourceModuleVersion: selected.moduleVersion,
			SourceLocation:      selected.location,
			TargetKind:          seed.TargetKind,
		}
		switch seed.TargetKind {
		case externals.TargetModule:
			target, found := targets[seed.SourcePackage+"\x00"+seed.SourceName]
			if !found {
				return nil, certifyError(
					"build binding",
					binding.SourceIdentity,
					"provider target is absent",
				)
			}
			binding.ModuleSpecifier = seed.ModuleSpecifier
			binding.Export = seed.Export
			binding.ImplementationOwner = target.owner
			binding.TargetFingerprint = target.fingerprint
		case externals.TargetSource:
			target, found := sources[seed.SourcePackage].functions[seed.TargetName]
			if !found || !target.body || !linkableSignature(target.signature) ||
				!types.Identical(selected.signature, target.signature) {
				return nil, certifyError(
					"build binding",
					binding.SourceIdentity,
					"portable source target is absent or incompatible",
				)
			}
			binding.TargetIdentity = target.contract.Identity()
			binding.TargetSignature = target.contract.Signature()
			binding.TargetLocation = target.location
		default:
			return nil, certifyError(
				"build binding",
				binding.SourceIdentity,
				"target kind is invalid",
			)
		}
		bindings = append(bindings, binding)
	}
	sort.Slice(bindings, func(left int, right int) bool {
		return bindings[left].SourceIdentity < bindings[right].SourceIdentity
	})
	integrity, err := providerDigest(resolved)
	if err != nil {
		return nil, err
	}
	return externals.Seal(externals.Document{
		SchemaVersion:                 externals.SchemaVersion,
		PackageName:                   externals.PackageName,
		PackageVersion:                providerPackage.Version,
		Backend:                       resolved.backend,
		GoVersion:                     resolved.buildProfile.ToolchainVersion(),
		GOOS:                          resolved.buildProfile.GOOS(),
		GOARCH:                        resolved.buildProfile.GOARCH(),
		CGOEnabled:                    resolved.buildProfile.CgoEnabled(),
		BuildTags:                     resolved.buildProfile.Tags(),
		ProviderIntegerRepresentation: providerIntegerRepresentation,
		StandardLibraryDigest:         standardLibraryDigest,
		ProviderDigest:                integrity,
		Bindings:                      bindings,
	})
}

func linkableSignature(signature *types.Signature) bool {
	return signature != nil && signature.Recv() == nil &&
		signature.TypeParams().Len() == 0 && !signature.Variadic()
}

func readStandardLibraryContract(
	manifestPath string,
	runtimePath string,
) (string, string, error) {
	payload, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", "", certifyError(
			"read standard-library manifest",
			manifestPath,
			err.Error(),
		)
	}
	manifest, err := gostdlib.Parse(payload)
	if err != nil {
		return "", "", err
	}
	canonical, err := gostdlib.Encode(manifest)
	if err != nil {
		return "", "", err
	}
	if !bytes.Equal(canonical, payload) {
		return "", "", certifyError(
			"read standard-library manifest",
			manifestPath,
			"manifest bytes are not canonical",
		)
	}
	if manifest.Digest() == "" {
		return "", "", fmt.Errorf("standard-library manifest digest is absent")
	}
	runtimePayload, err := os.ReadFile(runtimePath)
	if err != nil {
		return "", "", certifyError(
			"read standard-library runtime contract",
			runtimePath,
			err.Error(),
		)
	}
	requirements, err := runtimecontract.Decode(runtimePayload)
	if err != nil {
		return "", "", certifyError(
			"read standard-library runtime contract",
			runtimePath,
			err.Error(),
		)
	}
	digest := sha256.Sum256(runtimePayload)
	if hex.EncodeToString(digest[:]) != manifest.RuntimeDigest() {
		return "", "", certifyError(
			"read standard-library runtime contract",
			runtimePath,
			"content digest does not match the standard-library manifest",
		)
	}
	return manifest.Digest(), requirements.ProviderProfile().String(), nil
}
