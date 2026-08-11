package load

import environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"

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

func NewBuildProfileForToolchain(
	toolchainVersion string,
	goos string,
	goarch string,
	cgoEnabled bool,
	tags []string,
) (BuildProfile, error) {
	return environmentcontract.NewBuildProfileForToolchain(
		toolchainVersion,
		goos,
		goarch,
		cgoEnabled,
		tags,
	)
}

func DefaultBuildProfile() BuildProfile {
	return environmentcontract.DefaultBuildProfile()
}

func resolveBuildProfile(
	selected BuildProfile,
	version string,
	defaultGOOS string,
	defaultGOARCH string,
) (BuildProfile, error) {
	if selected.IsZero() {
		return NewBuildProfileForToolchain(
			version,
			defaultGOOS,
			defaultGOARCH,
			false,
			nil,
		)
	}
	if !selected.Valid() {
		return BuildProfile{}, &BuildProfileError{Reason: "profile is invalid"}
	}
	if selected.ToolchainVersion() != version {
		return BuildProfile{}, &BuildProfileError{Reason: "profile and selected Go tool differ"}
	}
	return selected, nil
}
