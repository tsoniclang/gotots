// Corpus translation: the profile's owned production packages loaded
// from the pinned checkout and translated as one unit.
package translate

import (
	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/typedload"
)

// Corpus loads the profile's owned production packages from the pinned
// checkout and translates them as one unit.
func Corpus(prof *profile.Profile, env []string, sourceDir string, options Options) (*Generated, error) {
	loaded, err := typedload.Load(prof, env, sourceDir)
	if err != nil {
		return nil, err
	}
	var unit []*packages.Package
	for _, p := range loaded {
		if typedload.RoleOf(p, sourceDir) != typedload.RoleProduction {
			continue
		}
		if class, _ := prof.Classify(p.PkgPath); class != profile.ClassOwned {
			continue
		}
		unit = append(unit, p)
	}
	return Packages(unit, sourceDir, options)
}
