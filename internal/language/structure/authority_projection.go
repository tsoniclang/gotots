package structure

import "github.com/tsoniclang/gotots/internal/identity"

type AuthorityProjection struct {
	id                 identity.PackageID
	localFiles         []identity.FileID
	certifiedFiles     []identity.FileID
	certifiedSynthetic bool
}

func (projection AuthorityProjection) ID() identity.PackageID {
	return projection.id
}

func (projection AuthorityProjection) LocalFiles() []identity.FileID {
	return append([]identity.FileID(nil), projection.localFiles...)
}

func (projection AuthorityProjection) CertifiedFiles() []identity.FileID {
	return append([]identity.FileID(nil), projection.certifiedFiles...)
}

func (projection AuthorityProjection) HasCertifiedSynthetic() bool {
	return projection.certifiedSynthetic
}

func (graph *Graph) AuthorityProjections() []AuthorityProjection {
	if graph == nil {
		return nil
	}
	out := make(
		[]AuthorityProjection, 0, len(graph.projections),
	)
	for index, projection := range graph.projections {
		localFiles := make(
			[]identity.FileID, 0, len(graph.packages[index].files),
		)
		for _, file := range graph.packages[index].files {
			localFiles = append(localFiles, file.Owner().ID().File())
		}
		certifiedFiles := make(
			[]identity.FileID, 0, len(projection.certifiedFiles),
		)
		for _, file := range projection.certifiedFiles {
			certifiedFiles = append(certifiedFiles, file.id)
		}
		out = append(out, AuthorityProjection{
			id:                 projection.id,
			localFiles:         localFiles,
			certifiedFiles:     certifiedFiles,
			certifiedSynthetic: projection.certifiedSynthetic,
		})
	}
	return out
}
