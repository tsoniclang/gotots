export declare class GoPointer<L, S> {
  readonly $go$address: object;
  static cell<L, S>(value: S): GoPointer<L, S>;
  static objectField<L, O extends object, K extends keyof O>(owner: O, key: K): GoPointer<L, O[K]>;
  static equal<LL, LS, RL, RS>(left: GoPointer<LL, LS> | undefined, right: GoPointer<RL, RS> | undefined): boolean;
  static dereference<L, S>(pointer: GoPointer<L, S> | undefined): GoPointer<L, S>;
  static direct<L>(pointer: L | undefined): L;
  static view<F, T, S>(pointer: GoPointer<F, S> | undefined): GoPointer<T, S> | undefined;
  get value(): S;
  set value(value: S);
}

export declare function goPointerHash<L, S>(pointer: GoPointer<L, S> | undefined): number;
