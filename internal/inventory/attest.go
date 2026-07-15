package inventory

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/pinning"
)

// attestBuildInputs proves that every file any inventoried package names —
// selected or ignored, Go or not, embed payloads included, in every
// partition class — is byte-identical to a blob in the pinned commit tree.
//
// This is the reverse direction of universe reconciliation: the universe
// proves every tracked file has a disposition; this proves every build
// input the toolchain can see is tracked and unmodified, so an ignored,
// build-tag-excluded, embedded, or non-Go input cannot escape attestation
// merely because it is never type-checked.
func attestBuildInputs(tree *pinning.Tree, packages []ModulePackage, sourceDir string) error {
	var problems []string
	verified := map[string]bool{}
	for i := range packages {
		for _, relative := range packages[i].buildInputFiles() {
			if verified[relative] {
				continue
			}
			verified[relative] = true
			data, err := os.ReadFile(filepath.Join(sourceDir, filepath.FromSlash(relative)))
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s: %v", packages[i].ImportPath, err))
				continue
			}
			if err := tree.VerifyFile(relative, data); err != nil {
				problems = append(problems, fmt.Sprintf("%s: %v", packages[i].ImportPath, err))
			}
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("build-input attestation fails closed on %d problems:\n%s",
			len(problems), strings.Join(problems, "\n"))
	}
	return nil
}
