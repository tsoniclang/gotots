// Hand port D3 — interface formatting tail: assertion helpers observe
// dynamic capability (KindString / Stringer) through TWO narrow
// interface probes, not the full concrete method universe. The 140x
// baseline expansion is the cost of serializing every member's complete
// method set into `any`; capability probing needs only these two.
export function AssertNever(member: GoAny, ...message: GoAny[]): void {
  let msg: string;
  if (message.length === 0) {
    msg = "Illegal value:";
  } else {
    msg = fmt.Sprint(...message);
  }
  let detail: string;
  const kindString = probe$KindString(member);
  const stringer = probe$Stringer(member);
  if (kindString !== undefined) {
    detail = kindString.m.KindString(kindString.v);
  } else if (stringer !== undefined) {
    detail = stringer.m.String(stringer.v);
  } else {
    detail = fmt.Sprintf("%v", member);
  }
  Fail(fmt.Sprintf("%s %s", msg, detail));
}
