# v0.2.0

## FEATURES

- `Walker`: `Walk` with a callback for each direction of travel. `Enter` fires on a struct-typed field before its children are walked, after any nil pointer has been initialized, and returns a `leave` that fires once the children and the field's own `Visit` have run. The ordering contract is enter → children → visit → leave, and the pairing is guaranteed: `leave` runs on every path out, including when a child errors, when `Enter` itself errors, and when any callback panics. That is what lets a caller bracket state around a subtree — a lexically scoped frame, say — without comparing paths or tracking how the walk left a field
- `EnterFunc`, the callback type for `Walker.Enter`

## IMPROVEMENTS

- `Walk` is now a one-line wrapper over `Walker`. Its signature and its behaviour are unchanged, including the existing rule that an error from a child skips the parent field's callback
- `UnpackType`'s doc comment no longer claims to strip interface wrappers. It only ever stripped pointers, and a `reflect.Type` describing an interface has no dynamic type behind it to unwrap

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