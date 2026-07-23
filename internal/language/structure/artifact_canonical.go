package structure

import (
	"sort"
)

func artifactFileIsCanonical(file artifactFile) bool {
	return sort.SliceIsSorted(
		file.Occurrences,
		func(i, j int) bool {
			return file.Occurrences[i].ID <
				file.Occurrences[j].ID
		},
	) &&
		sort.StringsAreSorted(file.Anchors) &&
		sort.SliceIsSorted(file.Definitions, func(i, j int) bool {
			return file.Definitions[i].ID <
				file.Definitions[j].ID
		}) &&
		sort.SliceIsSorted(file.Sites, func(i, j int) bool {
			return file.Sites[i].Definition <
				file.Sites[j].Definition
		}) &&
		sort.SliceIsSorted(file.Headers, func(i, j int) bool {
			return file.Headers[i].ID < file.Headers[j].ID
		}) &&
		sort.SliceIsSorted(file.Boundaries, func(i, j int) bool {
			return file.Boundaries[i].ID <
				file.Boundaries[j].ID
		}) &&
		sort.SliceIsSorted(file.Mappings, func(i, j int) bool {
			return file.Mappings[i].Definition <
				file.Mappings[j].Definition
		}) &&
		sort.SliceIsSorted(
			file.Owner.Directives,
			func(i, j int) bool {
				left := file.Owner.Directives[i]
				right := file.Owner.Directives[j]
				if left.Span.Start.Offset != right.Span.Start.Offset {
					return left.Span.Start.Offset <
						right.Span.Start.Offset
				}
				if left.Span.End.Offset != right.Span.End.Offset {
					return left.Span.End.Offset <
						right.Span.End.Offset
				}
				return left.Kind < right.Kind
			},
		)
}

func artifactPackageIsCanonical(pkg artifactPackage) bool {
	return sort.StringsAreSorted(pkg.Files) &&
		sort.StringsAreSorted(pkg.Owners) &&
		sort.SliceIsSorted(pkg.Definitions, func(i, j int) bool {
			return pkg.Definitions[i].ID <
				pkg.Definitions[j].ID
		}) &&
		sort.SliceIsSorted(pkg.Sites, func(i, j int) bool {
			return pkg.Sites[i].Definition <
				pkg.Sites[j].Definition
		}) &&
		sort.SliceIsSorted(pkg.Headers, func(i, j int) bool {
			return pkg.Headers[i].ID < pkg.Headers[j].ID
		}) &&
		sort.SliceIsSorted(pkg.Boundaries, func(i, j int) bool {
			return pkg.Boundaries[i].ID <
				pkg.Boundaries[j].ID
		})
}

func canonicalizeArtifactFile(file *artifactFile) {
	sort.Slice(file.Occurrences, func(i, j int) bool {
		return file.Occurrences[i].ID < file.Occurrences[j].ID
	})
	sort.Strings(file.Anchors)
	sort.Slice(file.Definitions, func(i, j int) bool {
		return file.Definitions[i].ID < file.Definitions[j].ID
	})
	sort.Slice(file.Sites, func(i, j int) bool {
		return file.Sites[i].Definition < file.Sites[j].Definition
	})
	sort.Slice(file.Headers, func(i, j int) bool {
		return file.Headers[i].ID < file.Headers[j].ID
	})
	sort.Slice(file.Boundaries, func(i, j int) bool {
		return file.Boundaries[i].ID < file.Boundaries[j].ID
	})
	sort.Slice(file.Mappings, func(i, j int) bool {
		return file.Mappings[i].Definition < file.Mappings[j].Definition
	})
	sort.Slice(file.Owner.Directives, func(i, j int) bool {
		left := file.Owner.Directives[i].Span
		right := file.Owner.Directives[j].Span
		if left.Start.Offset != right.Start.Offset {
			return left.Start.Offset < right.Start.Offset
		}
		if left.End.Offset != right.End.Offset {
			return left.End.Offset < right.End.Offset
		}
		return file.Owner.Directives[i].Kind <
			file.Owner.Directives[j].Kind
	})
}

func canonicalizeArtifactPackage(pkg *artifactPackage) {
	sort.Strings(pkg.Files)
	sort.Strings(pkg.Owners)
	sort.Slice(pkg.Definitions, func(i, j int) bool {
		return pkg.Definitions[i].ID < pkg.Definitions[j].ID
	})
	sort.Slice(pkg.Sites, func(i, j int) bool {
		return pkg.Sites[i].Definition < pkg.Sites[j].Definition
	})
	sort.Slice(pkg.Headers, func(i, j int) bool {
		return pkg.Headers[i].ID < pkg.Headers[j].ID
	})
	sort.Slice(pkg.Boundaries, func(i, j int) bool {
		return pkg.Boundaries[i].ID < pkg.Boundaries[j].ID
	})
}
