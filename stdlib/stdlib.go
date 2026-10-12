// Package stdlib registers Kavun's builtin modules. Each module is its own package under stdlib/ — its functions,
// the types it owns and its helpers — and registers itself (core.NewModule) when imported; importing this package
// imports them all. The kavun package imports it, so an embedder using kavun gets every module.
package stdlib

import (
	_ "github.com/jokruger/kavun/stdlib/base64"
	_ "github.com/jokruger/kavun/stdlib/clock"
	_ "github.com/jokruger/kavun/stdlib/exec"
	_ "github.com/jokruger/kavun/stdlib/fin"
	_ "github.com/jokruger/kavun/stdlib/fs"
	_ "github.com/jokruger/kavun/stdlib/hex"
	_ "github.com/jokruger/kavun/stdlib/io"
	_ "github.com/jokruger/kavun/stdlib/json"
	_ "github.com/jokruger/kavun/stdlib/math"
	_ "github.com/jokruger/kavun/stdlib/os"
	_ "github.com/jokruger/kavun/stdlib/path"
	_ "github.com/jokruger/kavun/stdlib/rand"
	_ "github.com/jokruger/kavun/stdlib/re"
)
