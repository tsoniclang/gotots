import { Seq, Seq2 } from "../../iter.js";

export class IterSeqValueOperations {
  static $project<T, Implementation>(source: Seq<T, Implementation>): Implementation {
    return source.value;
  }

  static $wrap<T, Implementation>(source: Implementation): Seq<T, Implementation> {
    return new Seq<T, Implementation>(source);
  }
}

export class IterSeq2ValueOperations {
  static $project<K, V, Implementation>(
    source: Seq2<K, V, Implementation>,
  ): Implementation {
    return source.value;
  }

  static $wrap<K, V, Implementation>(
    source: Implementation,
  ): Seq2<K, V, Implementation> {
    return new Seq2<K, V, Implementation>(source);
  }
}
