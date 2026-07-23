package source

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

// HydrationRequest is the validated structural-source demand presented to the
// sole semantic loader. It carries no provider or evidence-depth policy.
type HydrationRequest struct {
	files     []identity.FileID
	synthetic []identity.PackageID
}

// NewHydrationRequest constructs a selective request. Files are physical
// source files selected for local structural extraction; synthetic packages
// require checked-view declarations such as cgo output.
func NewHydrationRequest(
	files []identity.FileID,
	synthetic []identity.PackageID,
) (HydrationRequest, error) {
	out := HydrationRequest{
		files:     append([]identity.FileID(nil), files...),
		synthetic: append([]identity.PackageID(nil), synthetic...),
	}
	sort.Slice(out.files, func(i, j int) bool {
		return out.files[i].String() < out.files[j].String()
	})
	sort.Slice(out.synthetic, func(i, j int) bool {
		return out.synthetic[i].String() < out.synthetic[j].String()
	})
	for index, file := range out.files {
		if file.IsZero() {
			return HydrationRequest{}, fmt.Errorf(
				"hydration request contains a zero file identity",
			)
		}
		if index > 0 && out.files[index-1] == file {
			return HydrationRequest{}, fmt.Errorf(
				"hydration request duplicates file %s", file,
			)
		}
	}
	for index, pkg := range out.synthetic {
		if pkg.IsZero() {
			return HydrationRequest{}, fmt.Errorf(
				"hydration request contains a zero package identity",
			)
		}
		if index > 0 && out.synthetic[index-1] == pkg {
			return HydrationRequest{}, fmt.Errorf(
				"hydration request duplicates synthetic package %s", pkg,
			)
		}
	}
	return out, nil
}

// FileIDs returns isolated requested file identities.
func (r HydrationRequest) FileIDs() []identity.FileID {
	return append([]identity.FileID(nil), r.files...)
}

// SyntheticPackages returns isolated checked-view package identities.
func (r HydrationRequest) SyntheticPackages() []identity.PackageID {
	return append([]identity.PackageID(nil), r.synthetic...)
}
