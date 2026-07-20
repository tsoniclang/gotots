// Hand port B10 — tracedTypeAdapter.Display, the manual-required
// exemplar: the defer/recover suppression around the final return maps
// to try/catch as a reviewed manual body, not an automatic-lowering
// target. Go source comments are retained as part of the source shape.
Display(): string {
  // Compute display text for types where it's valuable for trace analysis.
  // TypeScript only does this for Anonymous|Literal types, but we extend to
  // unions, intersections, and template literals since they often lack
  // firstDeclaration and the display text helps identify them.
  // Incomplete types during tracing can cause panics, which we intentionally
  // suppress (returning ""), matching TypeScript's try/catch around typeToString.
  if (this.checker === undefined) {
    return "";
  }
  if ((this.t.objectFlags & ObjectFlagsAnonymous) !== 0 ||
    (this.t.flags & (TypeFlagsLiteral | TypeFlagsTemplateLiteral | TypeFlagsUnion | TypeFlagsIntersection)) !== 0) {
    try {
      return this.checker.TypeToString(this.t);
    } catch {
      return "";
    }
  }
  return "";
}
