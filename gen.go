package kavun

// The tables that register Kavun's native functions — each type's Methods table, the free builtin functions and each
// builtin module's registration — are generated from the `//kavun:` directives above their Go bodies into one
// kavun_gen.go per package (see cmd/kavun-gen). The same run generates kavun_reference.json, the language reference
// as data (see package reference), from the directives' doc comments; cmd/kavun-docs then renders the reference
// pages under docs/reference/ from kavun_reference.json alone. The doc comment is the documentation:
// kavun_reference.json and docs/reference/ are never edited by hand. Run `go generate ./...` (or `just generate`)
// after changing a directive or its doc comment; TestGeneratedInSync (which checks both stages with the flags of
// the lines below) and `just generate-check` fail while a generated file is out of date. The -docs-url value below
// is also the Justfile's docs_url: change both together.

//go:generate go run ./cmd/kavun-gen -docs-url https://github.com/jokruger/kavun/blob/main/docs ./...
//go:generate go run ./cmd/kavun-docs -out docs/reference kavun_reference.json
