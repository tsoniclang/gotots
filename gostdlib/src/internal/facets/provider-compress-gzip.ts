import {
  DirectGzipReader,
  type ProviderFlateReader,
  type ProviderReadCloser,
} from "./provider-compress-gzip-direct.js";
import type { ProviderErrorInterface } from "./provider-error.js";

export class GzipReaderOperations {
  static $copy<
    FlateReader extends ProviderFlateReader<Failure>,
    Failure extends ProviderErrorInterface,
    ReadCloser extends ProviderReadCloser<Failure>,
  >(
    source: DirectGzipReader<FlateReader, Failure, ReadCloser>,
  ): DirectGzipReader<FlateReader, Failure, ReadCloser> {
    return DirectGzipReader.$copy(source);
  }

  static $assign<
    FlateReader extends ProviderFlateReader<Failure>,
    Failure extends ProviderErrorInterface,
    ReadCloser extends ProviderReadCloser<Failure>,
  >(
    target: DirectGzipReader<FlateReader, Failure, ReadCloser>,
    source: DirectGzipReader<FlateReader, Failure, ReadCloser>,
  ): void {
    DirectGzipReader.$assign(target, source);
  }
}
