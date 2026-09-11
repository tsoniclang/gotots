package slice

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) sourceCountMethod(method, field Member) tsgo.MethodDeclaration {
	return b.method(nil, MemberName(method), nil, nil, b.integerInputType(), b.returnStatement(b.thisProperty(MemberName(field))))
}

func (b projectionBuilder) sourceCountMethod(member Member) tsgo.MethodDeclaration {
	return b.method(MemberName(member), nil, b.integerInputType(), b.returnStatement(b.call(b.source(), MemberName(member))))
}
