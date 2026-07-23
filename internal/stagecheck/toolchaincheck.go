package stagecheck

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tsoniclang/gotots/internal/source"
)

var independentToolchainKeys = []string{
	"GOROOT", "GOVERSION", "GOOS", "GOARCH", "GOEXPERIMENT",
	"GOFLAGS", "CGO_ENABLED", "CC", "CXX", "CGO_CFLAGS",
	"CGO_CPPFLAGS", "CGO_CXXFLAGS", "CGO_FFLAGS", "CGO_LDFLAGS",
	"PKG_CONFIG", "GOAMD64", "GOARM", "GOARM64", "GOMIPS",
	"GOMIPS64", "GOPPC64", "GORISCV64", "GOWASM",
}

func verifyToolchainEvidence(
	workspace *source.Workspace,
	request source.Request,
) error {
	if workspace == nil {
		return fmt.Errorf("workspace is absent")
	}
	recorded := workspace.Toolchain()
	if recorded.Binary() == "" || !filepath.IsAbs(recorded.Binary()) {
		return fmt.Errorf("selected toolchain path is not absolute")
	}
	binary, err := os.ReadFile(recorded.Binary())
	if err != nil {
		return fmt.Errorf("selected toolchain binary unreadable: %w", err)
	}
	if digest := fmt.Sprintf("%x", sha256.Sum256(binary)); digest !=
		recorded.BinaryDigest() {
		return fmt.Errorf("selected toolchain binary digest mismatch")
	}
	environment := append(os.Environ(), request.Env...)
	environment = append(
		environment,
		"PATH="+filepath.Dir(recorded.Binary())+
			string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	arguments := append([]string{"env"}, independentToolchainKeys...)
	command := exec.Command(recorded.Binary(), arguments...)
	command.Dir = request.Dir
	command.Env = environment
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("independent go env failed: %w", err)
	}
	lines := strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")
	if len(lines) != len(independentToolchainKeys) {
		return fmt.Errorf(
			"independent go env returned %d values, want %d",
			len(lines), len(independentToolchainKeys),
		)
	}
	values := map[string]string{}
	configuration := sha256.New()
	for index, key := range independentToolchainKeys {
		value := strings.TrimSpace(lines[index])
		values[key] = value
		fmt.Fprintf(configuration, "%s=%s\n", key, value)
	}
	expected := []struct {
		name     string
		recorded string
		derived  string
	}{
		{"GOROOT", recorded.GOROOT(), values["GOROOT"]},
		{"GOVERSION", recorded.Version(), values["GOVERSION"]},
		{"GOOS", recorded.GOOS(), values["GOOS"]},
		{"GOARCH", recorded.GOARCH(), values["GOARCH"]},
		{"GOEXPERIMENT", recorded.Experiments(), values["GOEXPERIMENT"]},
		{"GOFLAGS", recorded.GoFlags(), values["GOFLAGS"]},
		{"CGO_ENABLED", recorded.CgoEnabled(), values["CGO_ENABLED"]},
		{
			"build configuration digest",
			recorded.BuildConfigurationDigest(),
			fmt.Sprintf("%x", configuration.Sum(nil)),
		},
	}
	for _, fact := range expected {
		if fact.recorded != fact.derived {
			return fmt.Errorf(
				"%s mismatch: recorded %q, independent %q",
				fact.name, fact.recorded, fact.derived,
			)
		}
	}
	return nil
}
