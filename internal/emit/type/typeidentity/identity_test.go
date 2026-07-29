package typeidentity

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strconv"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestLocalComponentsIncludesInterfaceMethodContracts(t *testing.T) {
	fileSet := token.NewFileSet()
	source, err := parser.ParseFile(fileSet, "source.go", `package local

func use() {
	type Local int32
	var _ interface {
		Take(Local)
		Return() Local
		}
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Defs:   make(map[*ast.Ident]types.Object),
		Scopes: make(map[ast.Node]*types.Scope),
		Types:  make(map[ast.Expr]types.TypeAndValue),
	}
	checked, err := new(types.Config).Check(
		"example.com/local",
		fileSet,
		[]*ast.File{source},
		info,
	)
	if err != nil {
		t.Fatal(err)
	}
	var local *types.TypeName
	var contract *types.Interface
	for identifier, object := range info.Defs {
		if identifier.Name == "Local" {
			local, _ = object.(*types.TypeName)
		}
	}
	ast.Inspect(source, func(node ast.Node) bool {
		target, ok := node.(*ast.InterfaceType)
		if !ok {
			return true
		}
		represented, ok := info.Types[target].Type.(*types.Interface)
		if ok {
			contract = represented
		}
		return true
	})
	if local == nil || contract == nil || local.Pkg() != checked {
		t.Fatal("interface-local-component fixture is incomplete")
	}
	components := LocalComponents(contract)
	if len(components) != 1 || components[0] != local {
		t.Fatalf("local components = %#v, want Local", components)
	}
	function := source.Decls[0].(*ast.FuncDecl)
	owner := info.Defs[function.Name].(*types.Func)
	root := info.Scopes[function.Type]
	firstKey, err := LexicalNamedObjectKey(
		local,
		api.MustSourceArtifactOwner(owner),
		root,
	)
	if err != nil {
		t.Fatal(err)
	}
	if firstKey == "" {
		t.Fatal("local declaration has an empty lexical identity")
	}
}

func TestLexicalNamedObjectIdentityUsesExactScopeNotSourceOffset(
	t *testing.T,
) {
	firstObject, firstOwner, firstRoot := checkedLexicalObject(
		t,
		"package local\n\nfunc Use() {\n\t{\n\t\ttype Local int32\n\t\tvar _ Local\n\t}\n}\n",
		"Use",
	)
	relocatedObject, relocatedOwner, relocatedRoot := checkedLexicalObject(
		t,
		"package local\n\n\n\n\nfunc Use() {\n\t{\n\t\ttype Local int32\n\t\tvar _ Local\n\t}\n}\n",
		"Use",
	)
	firstKey, err := LexicalNamedObjectKey(
		firstObject,
		api.MustSourceArtifactOwner(firstOwner),
		firstRoot,
	)
	if err != nil {
		t.Fatal(err)
	}
	relocatedKey, err := LexicalNamedObjectKey(
		relocatedObject,
		api.MustSourceArtifactOwner(relocatedOwner),
		relocatedRoot,
	)
	if err != nil {
		t.Fatal(err)
	}
	if firstKey != relocatedKey {
		t.Fatalf(
			"source-offset-only mutation changed lexical identity: %s != %s",
			firstKey,
			relocatedKey,
		)
	}
	fabricatedOwner := types.NewFunc(
		token.NoPos,
		firstOwner.Pkg(),
		firstOwner.Name(),
		firstOwner.Type().(*types.Signature),
	)
	if _, err := LexicalNamedObjectKey(
		firstObject,
		api.MustSourceArtifactOwner(fabricatedOwner),
		firstRoot,
	); err == nil {
		t.Fatal("fabricated same-shape source artifact owner was accepted")
	}

	fileSet := token.NewFileSet()
	source, err := parser.ParseFile(fileSet, "source.go", `package local

func First(flag bool) {
	if flag {
		type Local int32
		var _ Local
	}
	if !flag {
		type Local int32
		var _ Local
	}
}

func Foreign() {
	type Local int32
	var _ Local
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Defs:   make(map[*ast.Ident]types.Object),
		Scopes: make(map[ast.Node]*types.Scope),
	}
	if _, err := new(types.Config).Check(
		"example.com/local",
		fileSet,
		[]*ast.File{source},
		info,
	); err != nil {
		t.Fatal(err)
	}
	firstFunction := source.Decls[0].(*ast.FuncDecl)
	foreignFunction := source.Decls[1].(*ast.FuncDecl)
	owner := info.Defs[firstFunction.Name].(*types.Func)
	artifactOwner := api.MustSourceArtifactOwner(owner)
	ownerRoot := info.Scopes[firstFunction.Type]
	foreignRoot := info.Scopes[foreignFunction.Type]
	var firstLocals []*types.TypeName
	var foreignLocal *types.TypeName
	for identifier, object := range info.Defs {
		local, ok := object.(*types.TypeName)
		if !ok || identifier.Name != "Local" {
			continue
		}
		switch {
		case scopeWithin(local.Parent(), ownerRoot):
			firstLocals = append(firstLocals, local)
		case scopeWithin(local.Parent(), foreignRoot):
			foreignLocal = local
		}
	}
	if len(firstLocals) != 2 || foreignLocal == nil {
		t.Fatal("lexical-identity mutation fixture is incomplete")
	}
	left, err := LexicalNamedObjectKey(
		firstLocals[0],
		artifactOwner,
		ownerRoot,
	)
	if err != nil {
		t.Fatal(err)
	}
	right, err := LexicalNamedObjectKey(
		firstLocals[1],
		artifactOwner,
		ownerRoot,
	)
	if err != nil {
		t.Fatal(err)
	}
	if left == right {
		t.Fatal("same-name sibling lexical declarations share one identity")
	}
	if _, err := LexicalNamedObjectKey(
		foreignLocal,
		artifactOwner,
		ownerRoot,
	); err == nil {
		t.Fatal("foreign same-name lexical declaration was accepted")
	}
}

func TestLexicalNamedObjectIdentityIncludesEnclosingDeclaration(
	t *testing.T,
) {
	fileSet := token.NewFileSet()
	source, err := parser.ParseFile(fileSet, "source.go", `package local

func First() {
	type Local int32
	var _ Local
}

func Second() {
	type Local int32
	var _ Local
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Defs:   make(map[*ast.Ident]types.Object),
		Scopes: make(map[ast.Node]*types.Scope),
	}
	if _, err := new(types.Config).Check(
		"example.com/local",
		fileSet,
		[]*ast.File{source},
		info,
	); err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, declaration := range source.Decls {
		function := declaration.(*ast.FuncDecl)
		owner := info.Defs[function.Name].(*types.Func)
		root := info.Scopes[function.Type]
		var local *types.TypeName
		for identifier, object := range info.Defs {
			candidate, ok := object.(*types.TypeName)
			if ok &&
				identifier.Name == "Local" &&
				scopeWithin(candidate.Parent(), root) {
				local = candidate
			}
		}
		if local == nil {
			t.Fatal("same-shaped owner fixture has no local declaration")
		}
		key, keyErr := LexicalNamedObjectKey(
			local,
			api.MustSourceArtifactOwner(owner),
			root,
		)
		if keyErr != nil {
			t.Fatal(keyErr)
		}
		keys = append(keys, key)
	}
	if len(keys) != 2 {
		t.Fatalf("lexical keys = %d, want 2", len(keys))
	}
	if keys[0] == keys[1] {
		t.Fatalf("distinct source artifacts share lexical key %q", keys[0])
	}
}

func checkedLexicalObject(
	t *testing.T,
	sourceText string,
	functionName string,
) (*types.TypeName, *types.Func, *types.Scope) {
	t.Helper()
	fileSet := token.NewFileSet()
	source, err := parser.ParseFile(
		fileSet,
		"source.go",
		sourceText,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Defs:   make(map[*ast.Ident]types.Object),
		Scopes: make(map[ast.Node]*types.Scope),
	}
	if _, err := new(types.Config).Check(
		"example.com/local",
		fileSet,
		[]*ast.File{source},
		info,
	); err != nil {
		t.Fatal(err)
	}
	var function *ast.FuncDecl
	for _, declaration := range source.Decls {
		candidate, ok := declaration.(*ast.FuncDecl)
		if ok && candidate.Name.Name == functionName {
			function = candidate
			break
		}
	}
	var local *types.TypeName
	for identifier, object := range info.Defs {
		if identifier.Name == "Local" {
			local, _ = object.(*types.TypeName)
		}
	}
	if function == nil || local == nil {
		t.Fatal("checked lexical-object fixture is incomplete")
	}
	owner, ok := info.Defs[function.Name].(*types.Func)
	if !ok {
		t.Fatal("checked lexical-object fixture has no function owner")
	}
	root := info.Scopes[function.Type]
	if root == nil {
		t.Fatal("checked lexical-object fixture is incomplete")
	}
	return local, owner, root
}

func scopeWithin(scope *types.Scope, root *types.Scope) bool {
	for current := scope; current != nil; current = current.Parent() {
		if current == root {
			return true
		}
	}
	return false
}

func TestParameterizedKeyUsesOwnerAssignedParameterIdentity(t *testing.T) {
	fileSet := token.NewFileSet()
	source, err := parser.ParseFile(fileSet, "source.go", `package generic

func First[T, U any](left T, right U) T { return left }
func Renamed[A, B any](left A, right B) A { return left }
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
	checked, err := new(types.Config).Check(
		"example.com/generic",
		fileSet,
		[]*ast.File{source},
		info,
	)
	if err != nil {
		t.Fatal(err)
	}
	first := checked.Scope().Lookup("First").Type().(*types.Signature)
	renamed := checked.Scope().Lookup("Renamed").Type().(*types.Signature)
	firstKey, err := BuildParameterizedKey(
		receiverFreeSignature(first),
		packageNamedIdentity,
		parameterOrdinalIdentity(first),
	)
	if err != nil {
		t.Fatal(err)
	}
	renamedKey, err := BuildParameterizedKey(
		receiverFreeSignature(renamed),
		packageNamedIdentity,
		parameterOrdinalIdentity(renamed),
	)
	if err != nil {
		t.Fatal(err)
	}
	if firstKey != renamedKey {
		t.Fatalf(
			"renaming equivalent type parameters changed identity: %s != %s",
			firstKey,
			renamedKey,
		)
	}
	_, err = BuildKey(receiverFreeSignature(first), packageNamedIdentity)
	if err == nil || !strings.Contains(err.Error(), "no identity owner") {
		t.Fatalf("unowned type parameter error = %v", err)
	}
}

func packageNamedIdentity(object *types.TypeName) (string, error) {
	return object.Pkg().Path() + "|" + object.Name(), nil
}

func parameterOrdinalIdentity(
	signature *types.Signature,
) TypeParameterIdentity {
	identities := make(map[*types.TypeParam]string)
	for index := range signature.TypeParams().Len() {
		identities[signature.TypeParams().At(index)] =
			"function|" + strconv.Itoa(index)
	}
	return func(parameter *types.TypeParam) (string, error) {
		return identities[parameter], nil
	}
}
