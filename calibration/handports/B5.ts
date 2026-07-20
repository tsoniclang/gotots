// Hand port B5 — address-taken field exception: the two fields live in
// cells because their addresses escape; the method itself stays ordinary.
getScope(privateName: boolean): GoCell<nameGenerationScope | undefined> {
  return core.IfElse(privateName, this.privateNameGenerationScope$cell, this.nameGenerationScope$cell);
}
