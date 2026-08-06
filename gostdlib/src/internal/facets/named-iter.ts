import { Seq, Seq2 } from "../../iter.js";

export class IterSeqValueOperations {
  static $project<T>(source: Seq<T>): Seq<T>["value"] {
    return source.value;
  }

  static $wrap<T>(source: Seq<T>["value"]): Seq<T> {
    return new Seq<T>(source);
  }
}

export class IterSeq2ValueOperations {
  static $project<K, V>(source: Seq2<K, V>): Seq2<K, V>["value"] {
    return source.value;
  }

  static $wrap<K, V>(source: Seq2<K, V>["value"]): Seq2<K, V> {
    return new Seq2<K, V>(source);
  }
}
