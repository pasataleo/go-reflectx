# v0.2.0

## FEATURES

- `Walker`: `Walk` with a callback for each direction of travel. `Enter` fires on every exported field before `Walk` recurses into it, including the fields it does not recurse into; `Visit` fires once the field's children have been walked. The ordering contract is enter → children → visit
- Both callbacks receive the field itself, with any nil pointer `Walk` is going to recurse into already initialized. A pointer `Walk` skips — a non-struct, or a back-reference to a type already on the current path — reaches `Enter` still nil, and stays that way
- `EnterFunc`, the callback type for `Walker.Enter`
- Data threading: `Enter` returns the data for its field, and that data is scoped to the field. The children below it and the field's own `Visit` receive it, while the field's siblings receive the data their shared parent produced. That is what lets a caller accumulate state over a subtree — a lexically scoped frame, say — building it in `Enter` and reading it back in `Visit`
- An error from `Enter` skips both the field's children and its `Visit`, and is aggregated alongside the rest

## IMPROVEMENTS

- `Walk` is now a one-line wrapper over `Walker`. Its behaviour is unchanged, including the existing rule that an error from a child skips the parent field's callback
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

## BREAKING CHANGES

- `WalkFunc` takes a fourth argument, the field's data
- `Walk` takes a `data` argument between the target and the callback. `Walk` builds a `Walker` with no `Enter`, so nothing derives a new data along the way and every callback receives what the caller passed in

## UPGRADE NOTES

- A caller with no use for the data passes `nil` and ignores the argument: `Walk(&target, callback)` becomes `Walk(&target, nil, callback)`, and the callback gains a trailing `_ interface{}` parameter