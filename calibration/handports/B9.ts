// Hand port B9 — value-receiver copy: the copy happens once at the
// method boundary; the call stays ordinary.
Move(dx: GoInt, dy: GoInt): GoInt {
  const p = this.goClone();
  p.X += dx;
  p.Y += dy;
  return p.X + p.Y;
}
