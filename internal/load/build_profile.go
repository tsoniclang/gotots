package load

import (
	"os"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
)

type BuildProfile = environmentcontract.BuildProfile
type BuildProfileError = environmentcontract.BuildProfileError

func NewBuildProfile(
	goos string,
	goarch string,
	cgoEnabled bool,
	tags []string,
) (BuildProfile, error) {
	return environmentcontract.NewBuildProfile(goos, goarch, cgoEnabled, tags)
}

func DefaultBuildProfile() BuildProfile {
	return environmentcontract.DefaultBuildProfile()
}

func resolveBuildProfile(selected BuildProfile) (BuildProfile, error) {
	if selected.IsZero() {
		return DefaultBuildProfile(), nil
	}
	if !selected.Valid() {
		return BuildProfile{}, &BuildProfileError{Reason: "profile is invalid"}
	}
	return selected, nil
}

func selectedBuildEnvironment(profile BuildProfile) []string {
	return profile.Environment(os.Environ())
}
