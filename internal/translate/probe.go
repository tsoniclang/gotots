package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/goid"
	"github.com/tsoniclang/gotots/internal/ir"
	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/typedload"
)

// ProbeResult measures how much of the real corpus the current reviewed
// subset translates, and ranks exactly what blocks the rest. It is
// diagnostic evidence for roadmap ordering — never authoritative
// generation, which requires complete package coverage.
type ProbeResult struct {
	Packages         int            `json:"packages"`
	Bodies           int            `json:"bodies"`
	Translated       int            `json:"translated"`
	Blocked          int            `json:"blocked"`
	BlockerHistogram map[string]int `json:"blockerHistogram"`
	// PackagesFullyTranslated lists packages where every production body
	// builds — candidates for end-to-end generation.
	PackagesFullyTranslated []string `json:"packagesFullyTranslated"`
	// PerPackage maps package path -> translated/total.
	PerPackage map[string]string `json:"perPackage"`
}

// Probe loads the owned corpus under the profile and attempts IR building
// for every production function body.
func Probe(prof *profile.Profile, env []string, sourceDir string) (*ProbeResult, error) {
	loaded, err := typedload.Load(prof, env, sourceDir)
	if err != nil {
		return nil, err
	}

	result := &ProbeResult{
		BlockerHistogram: map[string]int{},
		PerPackage:       map[string]string{},
	}
	sort.Slice(loaded, func(i, j int) bool { return loaded[i].ID < loaded[j].ID })

	for _, p := range loaded {
		if typedload.RoleOf(p, sourceDir) != typedload.RoleProduction {
			continue
		}
		class, _ := prof.Classify(p.PkgPath)
		if class != profile.ClassOwned {
			continue
		}
		result.Packages++
		packageBodies, packageTranslated := 0, 0

		for _, file := range p.Syntax {
			filename := p.Fset.Position(file.Pos()).Filename
			relative, relErr := filepath.Rel(sourceDir, filename)
			if relErr != nil || strings.HasPrefix(relative, "..") {
				continue
			}
			source, readErr := os.ReadFile(filename)
			if readErr != nil {
				return nil, readErr
			}
			for _, decl := range file.Decls {
				funcDecl, isFunc := decl.(*ast.FuncDecl)
				if !isFunc || funcDecl.Body == nil {
					continue
				}
				result.Bodies++
				packageBodies++
				if err := probeFunc(p, sourceDir, source, funcDecl); err != nil {
					result.Blocked++
					result.BlockerHistogram[blockerKey(err)]++
				} else {
					result.Translated++
					packageTranslated++
				}
			}
		}
		if packageBodies > 0 {
			result.PerPackage[p.PkgPath] = fmt.Sprintf("%d/%d", packageTranslated, packageBodies)
			if packageTranslated == packageBodies {
				result.PackagesFullyTranslated = append(result.PackagesFullyTranslated, p.PkgPath)
			}
		}
	}
	sort.Strings(result.PackagesFullyTranslated)
	return result, nil
}

func probeFunc(p *packages.Package, sourceDir string, source []byte, decl *ast.FuncDecl) error {
	start := p.Fset.Position(decl.Body.Pos()).Offset
	end := p.Fset.Position(decl.Body.End()).Offset
	if start < 0 || end > len(source) || start >= end {
		return fmt.Errorf("invalid body span")
	}
	digest := sha256.Sum256(source[start:end])
	id := goid.Func(p.PkgPath, decl.Name.Name)
	if decl.Recv != nil {
		id = goid.Method(p.PkgPath, receiverBase(decl.Recv), decl.Name.Name)
	}
	_, err := ir.BuildFunc(p, sourceDir, decl, id, hex.EncodeToString(digest[:]))
	return err
}

// blockerKey normalizes a diagnostic to its construct class so the
// histogram ranks semantic families, not source locations.
func blockerKey(err error) string {
	var unsupported *ir.Unsupported
	if ok := asUnsupported(err, &unsupported); ok {
		construct := unsupported.Construct
		// Strip per-site payloads (type spellings, identifiers) after the
		// stable class prefix for aggregation.
		for _, prefix := range []string{
			"non-basic type ", "basic type ", "pointer to non-named type ",
			"pointer to non-struct type ", "pointer to type outside the translated package: ",
			"map key type ", "type ", "identifier ", "call of ", "field access on ",
			"index on ", "nil comparison on ", "operator ", "conversion from ",
			"len of ", "make of ", "builtin ", "constant of type ", "zero value of ",
			"nil of type ", "equality on ", "ordering on ", "inc/dec of ",
			"struct type ", "composite literal of ", "index",
		} {
			if strings.HasPrefix(construct, prefix) {
				construct = strings.TrimSpace(prefix)
				break
			}
		}
		return unsupported.Code + ": " + construct
	}
	return "error: " + firstLine(err.Error())
}

func asUnsupported(err error, target **ir.Unsupported) bool {
	for err != nil {
		if u, ok := err.(*ir.Unsupported); ok {
			*target = u
			return true
		}
		unwrapper, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = unwrapper.Unwrap()
	}
	return false
}

func firstLine(s string) string {
	if index := strings.IndexByte(s, '\n'); index >= 0 {
		return s[:index]
	}
	return s
}
