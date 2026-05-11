# go-reflectx

Utilities for working with Go's `reflect` package.

## Installation

```sh
go get github.com/pasataleo/go-reflectx
```

## Features

### Walk

Recursively traverse a struct's exported fields with a callback. Handles pointer chains, auto-initializes nil pointers, detects self-referential types, and visits children before parents so nested structs are already populated when the parent callback fires.

### Path

A `[]string` type representing a field's location in a struct hierarchy (e.g. `Database.Host`), with `Append` and dot-separated `String()`.

### SetString

Populate a `reflect.Value` from a string. Supports a `StringSettable` interface for custom parsing, plus built-in handling for all primitive types (string, bool, int/uint/float/complex variants). Includes `CanSetString` to check support and an `EmptyStringIsTrue` option for flag-style booleans.

### Unpack / UnpackType

Strip pointer and interface wrappers from `reflect.Value` and `reflect.Type`, initializing nil pointers along the way.
