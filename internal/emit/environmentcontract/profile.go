package environmentcontract

import (
	"slices"

	environmentidentity "github.com/tsoniclang/gotots/internal/contracts/environment"
	externalcertify "github.com/tsoniclang/gotots/internal/contracts/externals/certify"
	gostdlibcertify "github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// ProfileError reports an invalid environment-profile input.
type ProfileError struct {
	Reason string
}

func (e *ProfileError) Error() string {
	return "environment profile: " + e.Reason
}

// Profile binds the settled environment evidence to the exact selected Go
// build profile, compilation profile, provider certificates, and pinned
// TS-Go target identity, each content-addressed by a deterministic
// fingerprint. It is deterministic proof evidence and is never read to
// drive emission.
type Profile struct {
	toolchainVersion       string
	goos                   string
	goarch                 string
	cgoEnabled             bool
	tags                   []string
	buildFingerprint       string
	integerRepresentation  api.IntegerRepresentation
	evaluationOrder        api.EvaluationOrder
	providerLinked         bool
	providerManifestDigest string
	externalLinked         bool
	externalManifestDigest string
	pinnedToolVersion      string
	schemaRevision         string
	schemaContractDigest   string
	protocolVersion        int
}

// NewProfile derives the complete exact profile identity. Provider and
// external certificate digests are stated with explicit absence: an unlinked
// certificate is represented by its linked flag, never by an unexplained
// empty digest.
func NewProfile(
	source *load.Program,
	integer api.IntegerRepresentation,
	order api.EvaluationOrder,
	standardLibrary *gostdlibcertify.Certificate,
	externalProvider *externalcertify.Certificate,
) (Profile, error) {
	providerLinked := standardLibrary != nil
	providerManifestDigest := ""
	if providerLinked {
		providerManifestDigest = standardLibrary.ManifestDigest()
	}
	externalLinked := externalProvider != nil
	externalManifestDigest := ""
	if externalLinked {
		externalManifestDigest = externalProvider.ManifestDigest()
	}
	if source == nil {
		return Profile{}, &ProfileError{Reason: "source program is nil"}
	}
	build := source.BuildProfile()
	if !build.Valid() {
		return Profile{}, &ProfileError{
			Reason: "build selection is invalid",
		}
	}
	buildFingerprint, err := environmentidentity.ToolchainKey(build)
	if err != nil {
		return Profile{}, err
	}
	if providerLinked && len(providerManifestDigest) != 64 {
		return Profile{}, &ProfileError{
			Reason: "linked provider certificate has no canonical manifest digest",
		}
	}
	if !providerLinked && providerManifestDigest != "" {
		return Profile{}, &ProfileError{
			Reason: "unlinked provider certificate carries a manifest digest",
		}
	}
	if externalLinked && len(externalManifestDigest) != 64 {
		return Profile{}, &ProfileError{
			Reason: "linked external certificate has no canonical manifest digest",
		}
	}
	if !externalLinked && externalManifestDigest != "" {
		return Profile{}, &ProfileError{
			Reason: "unlinked external certificate carries a manifest digest",
		}
	}
	return Profile{
		toolchainVersion:       build.ToolchainVersion(),
		goos:                   build.GOOS(),
		goarch:                 build.GOARCH(),
		cgoEnabled:             build.CgoEnabled(),
		tags:                   build.Tags(),
		buildFingerprint:       buildFingerprint,
		integerRepresentation:  integer,
		evaluationOrder:        order,
		providerLinked:         providerLinked,
		providerManifestDigest: providerManifestDigest,
		externalLinked:         externalLinked,
		externalManifestDigest: externalManifestDigest,
		pinnedToolVersion:      tsgo.PinnedToolVersion(),
		schemaRevision:         tsgo.PinnedSchemaRevision(),
		schemaContractDigest:   tsgo.PinnedSchemaContractDigest(),
		protocolVersion:        tsgo.PinnedProtocolVersion(),
	}, nil
}

func (p Profile) ToolchainVersion() string {
	return p.toolchainVersion
}

func (p Profile) GOOS() string {
	return p.goos
}

func (p Profile) GOARCH() string {
	return p.goarch
}

func (p Profile) CgoEnabled() bool {
	return p.cgoEnabled
}

func (p Profile) Tags() []string {
	return slices.Clone(p.tags)
}

// BuildFingerprint content-addresses the complete selected Go build profile.
func (p Profile) BuildFingerprint() string {
	return p.buildFingerprint
}

func (p Profile) IntegerRepresentation() api.IntegerRepresentation {
	return p.integerRepresentation
}

func (p Profile) EvaluationOrder() api.EvaluationOrder {
	return p.evaluationOrder
}

// ProviderManifestDigest is the certified gostdlib manifest digest. The
// second result is false when no provider certificate is linked.
func (p Profile) ProviderManifestDigest() (string, bool) {
	return p.providerManifestDigest, p.providerLinked
}

// ExternalManifestDigest is the certified external-provider manifest digest.
// The second result is false when no external provider is linked.
func (p Profile) ExternalManifestDigest() (string, bool) {
	return p.externalManifestDigest, p.externalLinked
}

func (p Profile) PinnedToolVersion() string {
	return p.pinnedToolVersion
}

// SchemaRevision is the exact pinned upstream TS-Go schema revision.
func (p Profile) SchemaRevision() string {
	return p.schemaRevision
}

// SchemaContractDigest content-addresses the complete pinned TS-Go schema
// contract manifest.
func (p Profile) SchemaContractDigest() string {
	return p.schemaContractDigest
}

func (p Profile) ProtocolVersion() int {
	return p.protocolVersion
}
