package kavun

// The tables that register Kavun's native functions — each type's Methods table, the free builtin functions and each
// builtin module's registration — are generated from the `//kavun:` directives above their Go bodies into one
// kavun_gen.go per package (see cmd/kavun-gen). The same run generates the reference pages under docs/reference/
// from the directives' doc comments: the doc comment is the documentation, docs/reference/ is never edited by hand.
// Run `go generate ./...` (or `just generate`) after changing a directive or its doc comment; TestGeneratedInSync
// fails while a generated file is out of date.

//go:generate go run ./cmd/kavun-gen ./...
