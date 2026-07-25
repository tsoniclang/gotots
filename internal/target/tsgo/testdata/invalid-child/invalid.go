package invalidchild

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func invalidChild(factory tsgo.Factory, statement tsgo.ExpressionStatement) {
	factory.VariableDeclaration(statement, nil, nil, nil)
}
