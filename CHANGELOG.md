# v0.2.0

## FEATURES

- `Walker`, a walk with an `Enter` callback before a field's children and a `Visit` callback after them.
- `Enter` can return data scoped to a field: its children and its `Visit` see it, its siblings do not.
- `EnterFunc`, the type of `Walker.Enter`.

## IMPROVEMENTS

- `Walk` is a thin wrapper over `Walker`, with unchanged behaviour.

<!--
## BUG FIXES
Issues that have been resolved.
-->

<!--
## SECURITY
Vulnerabilities or security-related changes addressed in this release.
-->

<!--
## DEPRECATIONS
Functionality that will be removed in a future release.
-->

## BREAKING CHANGES

- `Walk` takes a `data` argument, and `WalkFunc` a fourth parameter for it.

## UPGRADE NOTES

- `Walk(&target, callback)` becomes `Walk(&target, nil, callback)`, and the callback gains a trailing `_ interface{}` parameter.

# v0.1.0

## FEATURES

- `Walk` for recursively traversing exported struct fields with a callback; handles pointer chains, nil pointer initialization, self-referential type detection, and children-first ordering
- `Path` type for representing a field's location in a struct hierarchy with dot-separated formatting
- `SetString` for populating a `reflect.Value` from a string with built-in support for all primitive types and a `StringSettable` interface for custom parsing
- `CanSetString` to check whether a value can be set from a string
- `EmptyStringIsTrue` option for flag-style boolean fields
- `Unpack` and `UnpackType` for stripping pointer and interface wrappers from values and types
