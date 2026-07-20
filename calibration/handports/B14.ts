// Hand port B14 — typed-nil interface: a box holding a typed nil pointer
// is distinct from the nil interface (undefined).
export function Describe(useTypedNil: boolean): readonly [string, boolean] {
  let d: describer | undefined;
  if (useTypedNil) {
    const n: node | undefined = undefined;
    d = node$box$describer(n);
  }
  if (d === undefined) {
    return ["nil-interface", false];
  }
  return [d.m.describe(d.v), true];
}
