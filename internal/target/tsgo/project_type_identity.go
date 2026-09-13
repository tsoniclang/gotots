package tsgo

type ProjectTypeIdentity struct {
	symbolID uint64
	nullable bool
}

func (identity ProjectTypeIdentity) IncludesNullish() bool {
	return identity.nullable
}

func (i ProjectTypeIdentity) Matches(target ProjectExport) bool {
	return i.symbolID != 0 && i.symbolID == target.typeSymbolID
}

func (p *ProjectInspection) projectTypeIdentity(
	typeID uint32,
) (ProjectTypeIdentity, error) {
	nonNullable, err := p.nonNullableType(typeID)
	if err != nil {
		return ProjectTypeIdentity{}, err
	}
	var symbol *symbolResponse
	if err := requestProjectJSON(
		p.client,
		"getSymbolOfType",
		getSymbolOfTypeParams{Snapshot: p.snapshot, Type: nonNullable},
		&symbol,
	); err != nil {
		return ProjectTypeIdentity{}, err
	}
	if symbol == nil || symbol.ID == 0 {
		return ProjectTypeIdentity{}, &ProjectInspectionError{
			Operation: "project type identity",
			Reason:    "parameter type has no symbol",
		}
	}
	return ProjectTypeIdentity{symbolID: symbol.ID, nullable: nonNullable != typeID}, nil
}

func (p *ProjectInspection) typeArguments(typeID uint32) ([]typeResponse, error) {
	var selected []typeResponse
	if err := requestProjectJSON(
		p.client,
		"getTypeArguments",
		getTypePropertyParams{
			Snapshot: p.snapshot,
			Project:  p.project,
			Type:     typeID,
		},
		&selected,
	); err != nil {
		return nil, err
	}
	return selected, nil
}
