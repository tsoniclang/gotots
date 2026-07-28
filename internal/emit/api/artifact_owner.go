package api

import "go/types"

type ArtifactOwner struct {
	source    types.Object
	generated *GeneratedArtifact
}

func SourceArtifactOwner(source types.Object) (ArtifactOwner, error) {
	if source == nil {
		return ArtifactOwner{}, &RootRequestError{
			Reason: "source artifact owner is nil",
		}
	}
	return ArtifactOwner{source: source}, nil
}

func GeneratedArtifactOwner(
	generated *GeneratedArtifact,
) (ArtifactOwner, error) {
	if !generated.Valid() {
		return ArtifactOwner{}, &RootRequestError{
			Reason: "generated artifact owner is invalid",
		}
	}
	return ArtifactOwner{generated: generated}, nil
}

func MustSourceArtifactOwner(source types.Object) ArtifactOwner {
	owner, err := SourceArtifactOwner(source)
	if err != nil {
		panic(err)
	}
	return owner
}

func MustGeneratedArtifactOwner(generated *GeneratedArtifact) ArtifactOwner {
	owner, err := GeneratedArtifactOwner(generated)
	if err != nil {
		panic(err)
	}
	return owner
}

func (o ArtifactOwner) Valid() bool {
	return (o.source != nil) != (o.generated != nil) &&
		(o.generated == nil || o.generated.Valid())
}

func (o ArtifactOwner) Source() (types.Object, bool) {
	return o.source, o.source != nil && o.generated == nil
}

func (o ArtifactOwner) Generated() (*GeneratedArtifact, bool) {
	return o.generated, o.generated != nil && o.source == nil &&
		o.generated.Valid()
}

func (o ArtifactOwner) Name() string {
	if source, ok := o.Source(); ok {
		return source.Name()
	}
	if generated, ok := o.Generated(); ok {
		return generated.TargetName()
	}
	return ""
}
