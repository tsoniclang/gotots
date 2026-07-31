export declare class GoUnsafePointer {
  static from<P>(value: P | undefined): GoUnsafePointer | undefined;
  static to<P>(value: GoUnsafePointer | undefined): P | undefined;
}
