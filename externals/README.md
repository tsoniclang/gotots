# `@gotots/externals`

This strict-ESM package implements certified native boundaries selected by
GoToTS. Public module paths mirror their Go import paths. The checked contract
binds each export to an exact Go module version, source signature, build
profile, target profile, provider source owner, and TS-Go symbol fingerprint.

The package is linked locally during product verification. Publishing is not
part of this repository phase.
