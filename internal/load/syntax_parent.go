package load

import (
	"fmt"
	"go/ast"
)

func buildSyntaxParents(files []File) (map[ast.Node]ast.Node, error) {
	parents := make(map[ast.Node]ast.Node)
	for _, file := range files {
		if file.syntax == nil {
			return nil, fmt.Errorf("source file has no syntax tree")
		}
		stack := make([]ast.Node, 0, 32)
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			if node == nil {
				stack = stack[:len(stack)-1]
				return true
			}
			if len(stack) != 0 {
				parents[node] = stack[len(stack)-1]
			}
			stack = append(stack, node)
			return true
		})
		if len(stack) != 0 {
			return nil, fmt.Errorf("source syntax parent stack did not close")
		}
	}
	return parents, nil
}
