# v0.1.0

## FEATURES

- `Walk` for recursively traversing exported struct fields with a callback; handles pointer chains, nil pointer initialization, self-referential type detection, and children-first ordering
- `Path` type for representing a field's location in a struct hierarchy with dot-separated formatting
- `SetString` for populating a `reflect.Value` from a string with built-in support for all primitive types and a `StringSettable` interface for custom parsing
- `CanSetString` to check whether a value can be set from a string
- `EmptyStringIsTrue` option for flag-style boolean fields
- `Unpack` and `UnpackType` for stripping pointer and interface wrappers from values and types

<!--
## IMPROVEMENTS
Enhancements to existing functionality.
-->

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

<!--
## BREAKING CHANGES
Changes that are not backwards compatible and require updates from consumers.
-->

<!--
## UPGRADE NOTES
Steps required when upgrading from a previous version.
-->
