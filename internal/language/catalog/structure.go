package catalog

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

// DefinitionScope is the closed lexical scope in which Stage 1 asks whether
// a construct owns an independently selectable implementation.
type DefinitionScope uint8

const (
	DefinitionScopeInvalid DefinitionScope = iota
	DefinitionScopePackage
	DefinitionScopeExecutable
)

func (s DefinitionScope) Valid() bool {
	return s == DefinitionScopePackage || s == DefinitionScopeExecutable
}

// DefinitionContext is the complete parent-assigned context consumed by the
// definition algebra. Declaration is nonzero only below a GenDecl and keeps
// equal ValueSpec shapes under const and var semantically distinct.
type DefinitionContext struct {
	scope       DefinitionScope
	declaration TokenKind
}

// NewDefinitionContext validates a parent-assigned definition context.
func NewDefinitionContext(
	scope DefinitionScope,
	declaration TokenKind,
) (DefinitionContext, error) {
	if !scope.Valid() {
		return DefinitionContext{}, fmt.Errorf("invalid definition scope %d", scope)
	}
	if declaration != TokenInvalid {
		switch declaration {
		case TokenCONST, TokenIMPORT, TokenTYPE, TokenVAR:
		default:
			return DefinitionContext{}, fmt.Errorf(
				"token %s is not a declaration class", declaration,
			)
		}
	}
	return DefinitionContext{scope: scope, declaration: declaration}, nil
}

func (c DefinitionContext) Scope() DefinitionScope { return c.scope }
func (c DefinitionContext) Declaration() TokenKind { return c.declaration }
func (c DefinitionContext) Valid() bool {
	_, err := NewDefinitionContext(c.scope, c.declaration)
	return err == nil
}

// WithDeclaration returns the context assigned by a GenDecl parent.
func (c DefinitionContext) WithDeclaration(
	declaration TokenKind,
) (DefinitionContext, error) {
	return NewDefinitionContext(c.scope, declaration)
}

// DefinitionKind classifies one construct through the catalog's sole
// definition algebra. hasExecutionEntry is derived from catalog edges. All
// parent-sensitive facts arrive in context; the child never inspects a parent.
func DefinitionKind(
	kind Kind,
	context DefinitionContext,
	hasExecutionEntry bool,
) (identity.DefinitionKind, bool, error) {
	if !kind.Valid() || !context.Valid() {
		return identity.DefinitionInvalid, false, fmt.Errorf(
			"invalid definition query kind=%s scope=%d declaration=%s",
			kind, context.scope, context.declaration,
		)
	}
	switch kind {
	case KindFuncDecl:
		if context.scope != DefinitionScopePackage {
			return identity.DefinitionInvalid, false, nil
		}
		if hasExecutionEntry {
			return identity.DefinitionFuncDecl, true, nil
		}
		return identity.DefinitionBodylessDecl, true, nil
	case KindFuncLit:
		if context.scope == DefinitionScopeExecutable {
			return identity.DefinitionFuncLit, true, nil
		}
	case KindValueSpec:
		if context.scope == DefinitionScopePackage &&
			context.declaration == TokenVAR &&
			hasExecutionEntry {
			return identity.DefinitionPackageInitializer, true, nil
		}
	}
	return identity.DefinitionInvalid, false, nil
}

// DefinitionEntry reports whether this edge enters executable content owned by
// a definition. The answer is derived from the edge's one grammatical role;
// there is no second field-name or edge-ID table.
func (e Edge) DefinitionEntry() bool {
	switch e.Role() {
	case RoleFunctionBody, RoleInitializerValue:
		return true
	default:
		return false
	}
}

// CarriesToken reports whether one construct kind owns a lexical token value
// that must be retained in its canonical occurrence payload.
func (k Kind) CarriesToken() bool {
	switch k {
	case KindBasicLit,
		KindUnaryExpr,
		KindBinaryExpr,
		KindIncDecStmt,
		KindAssignStmt,
		KindBranchStmt,
		KindGenDecl:
		return true
	default:
		return false
	}
}
