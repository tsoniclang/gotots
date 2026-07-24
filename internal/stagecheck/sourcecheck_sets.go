package stagecheck

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/source"
)

func stringSet(values []string) map[string]bool {
	set := map[string]bool{}
	for _, value := range values {
		set[value] = true
	}
	return set
}

func fileIDSet(pkg *source.Package) map[identity.FileID]bool {
	set := map[identity.FileID]bool{}
	for _, file := range pkg.Files() {
		set[file.ID()] = true
	}
	return set
}

func inputSet(pkg *source.Package) map[string]bool {
	set := map[string]bool{}
	for _, input := range pkg.Inputs() {
		set[fmt.Sprintf(
			"%s|%s|%s|%t",
			input.Kind(),
			input.ID(),
			input.ByteDigest(),
			input.Overlaid(),
		)] = true
	}
	return set
}

func joinSet(
	id identity.PackageID,
	class string,
	got, want map[string]bool,
	problems *problemSet,
) {
	for member := range got {
		if !want[member] {
			problems.addf(
				"%s holds %s %s the toolchain does not name",
				id, class, member,
			)
		}
	}
	for member := range want {
		if !got[member] {
			problems.addf(
				"%s misses toolchain %s %s", id, class, member,
			)
		}
	}
}

func joinFileIDSet(
	id identity.PackageID,
	class string,
	got, want map[identity.FileID]bool,
	problems *problemSet,
) {
	for member := range got {
		if !want[member] {
			problems.addf(
				"%s holds %s %s the toolchain does not name",
				id, class, member,
			)
		}
	}
	for member := range want {
		if !got[member] {
			problems.addf(
				"%s misses toolchain %s %s", id, class, member,
			)
		}
	}
}
