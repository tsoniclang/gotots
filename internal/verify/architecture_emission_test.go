package verify

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

func verifyEmissionSource(
	relative string,
	file *ast.File,
	importAliases map[string]string,
) error {
	for _, forbiddenImport := range []string{
		"go/format",
		"go/parser",
		"go/printer",
		"html/template",
		"text/template",
	} {
		if importAliases[forbiddenImport] != "" {
			return &wallError{
				source: relative,
				reason: "emission may construct only typed TS-Go AST",
			}
		}
	}
	astAlias := importAliases["go/ast"]
	interfaceValueAlias := importAliases[modulePath+"/internal/emit/value/interfacevalue"]
	var violation string
	ast.Inspect(file, func(node ast.Node) bool {
		if violation != "" {
			return false
		}
		switch node := node.(type) {
		case *ast.CallExpr:
			selector, ok := node.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if selector.Sel.Name == "ClassDeclaration" &&
				relative != "internal/emit/typescriptclass/contract.go" {
				violation = "generated class bypasses the Promise-assimilation owner"
				return false
			}
			if selector.Sel.Name == "ThrowStatement" &&
				relative != "internal/emit/runtime/panic/owner.go" {
				violation = "target throw is owned only by the panic runtime"
				return false
			}
			if selector.Sel.Name == "NonNullExpression" {
				violation = "unchecked target non-null assertion is forbidden"
				return false
			}
			qualifier, qualifierOK := selector.X.(*ast.Ident)
			if qualifierOK &&
				qualifier.Name == interfaceValueAlias &&
				(selector.Sel.Name == "AdaptExpected" ||
					(selector.Sel.Name == "Assign" &&
						relative != "internal/emit/value/representation/owner.go")) {
				violation = "interface transfer bypasses the value-transfer owner"
				return false
			}
			for _, forbidden := range forbiddenInterfaceBoundarySelectors(relative) {
				if selector.Sel.Name == forbidden {
					violation = "interface boundary bypasses its method-token owner with " +
						forbidden
					return false
				}
			}
			if qualifierOK && qualifier.Name == astAlias &&
				(selector.Sel.Name == "Walk" || selector.Sel.Name == "Inspect") {
				violation = "production emission uses generic AST recursion"
			}
		case *ast.BasicLit:
			if node.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(node.Value)
			if err != nil {
				violation = "invalid production string literal"
				return false
			}
			if value == "then" &&
				relative != "internal/emit/typescriptclass/contract.go" &&
				relative != "internal/emit/runtime/scheduler/owner.go" {
				violation = "Promise-assimilation spelling bypasses its class or scheduler owner"
				return false
			}
			if relative != "internal/emit/api/artifact_name_contract.go" && strings.HasPrefix(value, "__gotots_") {
				violation = "generated temporary spelling bypasses the canonical name owner"
				return false
			}
			for _, forbidden := range []string{
				".apply(",
				".bind(",
				".call(",
				"as any",
				"as unknown",
				"import(",
				"module.exports",
				"require(",
				"/// <reference",
			} {
				if strings.Contains(value, forbidden) {
					violation = "production emission contains forbidden target fragment " + forbidden
					return false
				}
			}
		}
		return true
	})
	if violation != "" {
		return &wallError{source: relative, reason: violation}
	}
	return nil
}

func forbiddenInterfaceBoundarySelectors(relative string) []string {
	switch relative {
	case "internal/emit/declaration/interfaceadapter/handler.go":
		return []string{
			"ValueContract",
			"SourceValueContract",
			"GeneratedValueCall",
			"SelectedMethodCall",
		}
	case "internal/emit/declaration/interfacetype/handler.go":
		return []string{"ValueContract", "SourceValueContract"}
	case "internal/emit/expression/call/method.go":
		return []string{
			"ValueContract",
			"ValueCall",
			"DetachedValueCall",
			"InterfaceMethodToken",
		}
	case "internal/emit/expression/call/deferred_method.go":
		return []string{"ValueContract", "ValueCall", "InterfaceMethodToken"}
	case "internal/emit/expression/methodvalue/handler.go",
		"internal/emit/expression/methodexpression/handler.go":
		return []string{"InterfaceMethodToken"}
	case "internal/emit/generic/capability/constraint_method.go":
		return []string{"ValueContract", "ValueCall", "InterfaceMethodToken"}
	default:
		return nil
	}
}
