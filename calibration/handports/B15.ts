// Hand port B15 — method value: the receiver is captured exactly once,
// at creation.
export function Capture(): readonly [GoInt, GoInt] {
  let c = new counter(1n);
  const receiver = c;
  const value = () => receiver.Value();
  c = new counter(100n);
  void c;
  const first = value();
  return [first, first];
}
