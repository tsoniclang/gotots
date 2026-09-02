package callableimplementation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	implementationcontract "github.com/tsoniclang/gotots/internal/contracts/implementation"
)

func TestStagedVerificationUsesOnlyCertifiedDeclarationSources(t *testing.T) {
	t.Run("executable source is rejected", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "runtime-contract.ts")
		if _, err := implementationcontract.NewCertificationSource(path, strings.Repeat("0", 64)); err == nil {
			t.Fatal("executable certification source was accepted")
		}
	})

	t.Run("ambient unknown declaration enables exact authored call", func(t *testing.T) {
		fixture := newStagedVerificationFixture(t)
		config, certificationPath := fixture.configWithCertificationSource(t)
		verified, err := VerifyStagedGeneratedContracts(config)
		if err != nil {
			t.Fatal(err)
		}
		if len(verified) != 1 {
			t.Fatalf("verified modules = %d, want 1", len(verified))
		}
		if _, err := os.Stat(certificationPath); err != nil {
			t.Fatalf("certification source was modified: %v", err)
		}
	})

	t.Run("ambient explicit any is rejected", func(t *testing.T) {
		fixture := newStagedVerificationFixture(t)
		config, certificationPath := fixture.configWithCertificationSource(t)
		payload := []byte(
			"declare module \"@fixture/runtime.js\" {\n" +
				"  export function identity(value: any): number;\n" +
				"}\n",
		)
		if err := replaceCertificationSource(config.Modules, certificationPath, payload); err != nil {
			t.Fatal(err)
		}
		_, err := VerifyStagedGeneratedContracts(config)
		if err == nil || !strings.Contains(err.Error(), "explicit-any") {
			t.Fatalf("certification any policy error = %v", err)
		}
	})

	t.Run("declaration digest drift fails", func(t *testing.T) {
		fixture := newStagedVerificationFixture(t)
		config, certificationPath := fixture.configWithCertificationSource(t)
		if err := os.WriteFile(
			certificationPath,
			[]byte("declare module \"@fixture/runtime.js\" {}\n"),
			0o600,
		); err != nil {
			t.Fatal(err)
		}
		_, err := VerifyStagedGeneratedContracts(config)
		if err == nil || !strings.Contains(err.Error(), "source digest changed") {
			t.Fatalf("certification source mutation error = %v", err)
		}
	})

	t.Run("declaration source policy fails", func(t *testing.T) {
		fixture := newStagedVerificationFixture(t)
		config, certificationPath := fixture.configWithCertificationSource(t)
		payload := []byte(
			"// @ts-nocheck\n" +
				"declare module \"@fixture/runtime.js\" {\n" +
				"  export function identity(value: number): number;\n" +
				"}\n",
		)
		if err := replaceCertificationSource(config.Modules, certificationPath, payload); err != nil {
			t.Fatal(err)
		}
		_, err := VerifyStagedGeneratedContracts(config)
		if err == nil || !strings.Contains(err.Error(), "diagnostic-suppression") {
			t.Fatalf("certification source policy error = %v", err)
		}
	})
}

func (f *stagedVerificationFixture) configWithCertificationSource(
	t *testing.T,
) (StagedVerificationConfig, string) {
	t.Helper()
	config := f.config(
		t,
		"import { identity } from \"@fixture/runtime.js\";\n"+
			"export function addFast(value: number): number { return identity(value); }\n",
		[]string{"addFast"},
	)
	certificationPath := filepath.Join(f.root, "runtime-contract.d.ts")
	payload := []byte(
		"declare module \"@fixture/runtime.js\" {\n" +
			"  export function identity(value: unknown): number;\n" +
			"}\n",
	)
	if err := os.WriteFile(certificationPath, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(payload)
	certificationSource, err := implementationcontract.NewCertificationSource(
		certificationPath,
		hex.EncodeToString(digest[:]),
	)
	if err != nil {
		t.Fatal(err)
	}
	module := config.Modules[0]
	module, err = NewStagedModule(
		module.sourcePath,
		module.outputPath,
		module.sourceDigest,
		module.exports,
		[]implementationcontract.CertificationSource{certificationSource},
	)
	if err != nil {
		t.Fatal(err)
	}
	config.Modules[0] = module
	return config, certificationPath
}

func replaceCertificationSource(
	modules []StagedModule,
	certificationPath string,
	payload []byte,
) error {
	if len(modules) != 1 {
		return fmt.Errorf("staged module denominator is %d, want 1", len(modules))
	}
	if err := os.WriteFile(certificationPath, payload, 0o600); err != nil {
		return err
	}
	digest := sha256.Sum256(payload)
	changed, err := implementationcontract.NewCertificationSource(
		certificationPath,
		hex.EncodeToString(digest[:]),
	)
	if err != nil {
		return err
	}
	module := modules[0]
	module, err = NewStagedModule(
		module.sourcePath,
		module.outputPath,
		module.sourceDigest,
		module.exports,
		[]implementationcontract.CertificationSource{changed},
	)
	if err != nil {
		return err
	}
	modules[0] = module
	return nil
}
