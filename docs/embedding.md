# Embedding Kavun In Go

The recommended embedding API is `kavun.Script`. It wraps parsing, compilation, bindings setup, and VM execution into a
higher-level workflow that is easier to integrate and maintain in Go applications.

For lower-level control over compilation and execution, direct use of the compiler and VM is still available; this
document focuses on the Script-first approach.

## Quick Start

The primary pattern: create a script, compile once, then run multiple times.

```go
package main

import (
	"fmt"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/vm"
)

func main() {
	src := []byte(`
fib := func(x) {
    if x < 2 {
        return x
    }
    return fib(x-1) + fib(x-2)
}
out = fib(10)
`)

	// Create and configure script
	script := kavun.NewScript(src, "out")

	// Create VM
	machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

	// Compile once
	compiled, err := script.Compile()
	if err != nil {
		panic(err)
	}

	// Run repeatedly with the same compiled code
	for i := 0; i < 100; i++ {
		if err := compiled.Run(machine); err != nil {
			panic(err)
		}
	}

	out, err := compiled.Get("out")
	if err != nil {
		panic(err)
	}
	fmt.Println("result:", out.String())
}
```

## Inputs and Outputs

Setup input/output bindings before compilation:

```go
script := kavun.NewScript(src, "x", "y", "out")
compiled, err := script.Compile()
```

Before each run, set input values with `compiled.Set(...)`:

```go
if err := compiled.Set("x", core.IntValue(50)); err != nil {
    panic(err)
}
if err := compiled.Set("y", core.IntValue(7)); err != nil {
    panic(err)
}
if err := compiled.Run(machine); err != nil {
    panic(err)
}
```

After execution, retrieve output values with `Get`:

```go
out, err := compiled.Get("out")
if err != nil {
    panic(err)
}
sum, _ := out.AsInt()
fmt.Println(sum)
```

### Converting Go values

`kavun.ValueOf(v)` converts a Go value into a `core.Value` (`kavun.MustValueOf` panics instead of returning the
error); `kavun.Eval` runs it on every parameter. It accepts exactly these Go types, recursing into containers, and
returns an error for any other:

| Go type | Kavun value |
|---|---|
| `nil` | `undefined` |
| `bool` | `bool` |
| `byte` (`uint8`) / `[]byte` | `byte` / `bytes` |
| `rune` (`int32`) / `[]rune` | `rune` / `runes` |
| `int`, `int64` | `int` |
| `float64` | `float` |
| `string` | `string` |
| `json.Number` | `int` or `float`, by `json.decode`'s rule (below) |
| `dec128.Dec128` | `decimal` |
| `time.Time` / `civil.Date` | `time` / `date` |
| `error` | `error` (kind `user`, message = `Error()`) |
| `[]string`, `[]any`, `[]core.Value` | `array` |
| `map[string]any`, `map[string]core.Value` | `record` |
| `core.NativeFunc` | `function` |
| `core.Value` | itself |

Other integer and float widths (`int8`, `uint`, `float32`, …) and pointers are not accepted: convert them first.

**Go functions.** A bare `core.NativeFunc` becomes a closure named `anonymous` that accepts any count (arity 0,
variadic — it renders `<builtin-closure:anonymous/0+>`): its body checks what it receives. To give a Go function a
name and a parameter list, build the closure yourself with a `core.Signature`; the call is then checked against it
before the body runs, exactly like a builtin's:

```go
var clampSig = core.MustSignature("(x, lo, hi)") // parsed once; a malformed template panics here

clamp := core.NewBuiltinClosureValue("clamp", clampSig, func(_ core.VM, args []core.Value) (core.Value, error) {
    x, _ := args[0].AsInt() // three arguments are guaranteed
    lo, _ := args[1].AsInt()
    hi, _ := args[2].AsInt()
    return core.IntValue(min(max(x, lo), hi)), nil
})
// clamp(1) in a script raises: wrong_num_arguments: (clamp) expected (x, lo, hi), got 1 argument
```

A template is the parameter list a script author would write: names, an optional group `[, x]`, a rest `...xs`, and
readings joined by ` | ` (see [conventions.md § Arity and templates](conventions.md#arity-and-templates)).
`core.ParseSignature(template)` returns the error instead of panicking; `core.SignatureOf(arity, variadic)` builds
the bare pair when only the count is known at run time (the error then states it: `(f) expected 2 arguments, got 1
argument`). A body whose template names an upper bound (`"(x[, y])"`) refuses more by returning `core.ErrArgCount`.
A closure answers its pair to callback-taking members like any function: `xs.map(clamp)` is refused (it accepts
three arguments, not one), so a script adapts it with a lambda (`xs.map(x => clamp(x, 0, 9))`).

**JSON a host decoded itself.** `encoding/json` decodes every number to `float64` by default, which silently
changes integers above 2⁵³. Decode with `UseNumber()` instead and pass the result as is: each `json.Number` becomes
the value `json.decode` would answer for the same text — `int` without a fraction or exponent, `float` otherwise —
and one that does not fit (`9223372036854775808`, `1e999`) is an error, never a rounded float. Text that is not a
JSON number (`"0x10"`, `" 1"`, `"NaN"`) is an error too.

```go
dec := json.NewDecoder(body)
dec.UseNumber()
var payload map[string]any
if err := dec.Decode(&payload); err != nil {
    return err
}
v, err := kavun.ValueOf(payload) // {"id": 9007199254740993} -> id is the int 9007199254740993, exact
```

### Encoding values

`core.Value.EncodeBinary` / `DecodeBinary` — and `encoding/gob`, which goes through them — carry **data**. A
function (builtin, member, script function) or a module value means something only inside one application and its
configuration, so the codec refuses it on encode and on decode, alone or anywhere inside a container, with a
`binary_encoding` error naming where:

```text
binary_encoding: dict value at key "f": value type function does not support binary encoding
```

Code travels only in bytecode.

## Modules and Imports

### The module grant

A builtin module either only computes (`math`, `json`, `fin`, … — *pure*) or touches the world (`clock`,
`exec`, `fs`, `io`, `os`). By default a script may import **only the pure modules** (`core.PureModuleNames()`); anything
else must be granted. The grant is checked twice — by the compiler, and again by the VM that runs the bytecode —
so bytecode compiled elsewhere under a wider grant cannot run under a narrower one. The bytecode lists every builtin module its
code can import (`Bytecode.Imports`, collected by the compiler, custom and file modules included); the VM checks
that list once, when the bytecode first meets it (`Reset`), and nothing is checked while the script runs. The list
is part of the bytecode, so the VM's check holds only for bytecode you trust — see "Bytecode" below:

```go
script.SetAllowedModules("math", "os") // compile-time gate: exactly these
compiled, err := script.Compile()

machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
if err := machine.SetAllowedModules("math", "os"); err != nil { // run-time gate: an unknown name is a host error
    return err
}
err = compiled.Run(machine)
```

`Script.AllowAllModules()` and `VM.AllowAllModules()` grant everything (the `kavun` CLI does both). A refusal at
compile time is a compile error (`module 'os' is not allowed to import`); in the VM, `Run` answers
`module_not_allowed` (`module 'os' is not allowed`) before the first instruction — none of the script runs, so it
cannot `recover()` the refusal, and an import the script would never reach still counts. The VM keeps its grant
across `Reset`. `kavun.Eval` runs with
the default grant. The **grant is the security boundary** for what a script may do to the host: grant world
modules only to scripts you trust with them.

The grant governs what a script's code may import. A value the host builds or obtains in Go and hands to the VM —
`Compiled.Set`, globals, `VM.Call` arguments, a builtin's result — is the host's own grant: a world module's
function passed in this way is callable whatever the VM's grant, and nothing checks it at call time.

### Standard streams

`VM.SetStdout(w)` and `VM.SetStdin(r)` set the streams the VM hands to modules that print or read (default
`os.Stdout` / `os.Stdin`); `VM.Stdout()` / `VM.Stdin()` answer them, also through the `core.VM` interface.

### Your own builtin module

An embedder registers its own builtin modules beside the standard ones. A builtin module is a type; an
embedder's lives in the embedder range (type ids 192–255, shared with embedder types). Declare its
functions and constants through `core.NewModule`, each under a member id (a builtin name, or one you bind with
`member.Define`), and a builtin-function slot 0–127:

```go
const myModule = uint8(200)

var (
    memberDouble = member.ID(2100) // embedder member ids start at member.FirstUserDefined (2048)
    memberLabel  = member.ID(2101)
)

func init() {
    core.SetHostVersion(1, 0) // required once you register types or modules: see "Bytecode" below
    member.Define(memberDouble, "double")
    member.Define(memberLabel, "label")
    err := core.NewModule(myModule, "mymod", true). // true: pure — granted by default
        PureFn(memberDouble, 0, "(n)", func(_ core.VM, args []core.Value) (core.Value, error) {
            n, _ := args[0].AsInt()
            return core.IntValue(2 * n), nil
        }).
        Const(memberLabel, core.NewStringValue("mine")).
        Register()
    if err != nil {
        panic(err)
    }
}
```

`PureFn` claims a function pure (the optimizer may fold its calls), `Fn` does not. The template is the count
check: a call is checked against it before the function runs (`(mymod.double) expected (n), got 0 arguments`), so
`args[0]` above is guaranteed; a function with an optional tail bounds it itself and refuses with
`core.ErrArgCount`. `New(f)` makes `import` answer a
fresh value per import instead of one singleton, `Type(typeID, descr)` registers a type the module owns (a handle or
record type its functions answer, in the embedder range like the module), and `Descr()` lets you amend the module
type's hooks before `Register`. `core.UnregisterModule(typeID)` removes an embedder module and its types again
(teardown, tests); `kavun.ModuleNames()` lists every registered module. The standard modules are built the same
way, one package each under `stdlib/` — `stdlib/fs` is a compact example of a module with an owned type.

### Imports from source

```go

// Enable/disable file imports
script.DisableFileImport()
script.EnableFileImport()

// In-memory source module
script.AddCustomModule("helpers", []byte(`
export {add: func(a, b) { return a + b }}
`))
```

## Bytecode: persisting, signing, compatibility

A compiled script can be stored and run later: `compiled.Encode(w, signer)` writes its code and its binding names
(not the values bound to them), and `kavun.DecodeCompiled(r, signer)` answers a `*Compiled` whose bindings start
undefined — `Set`, `Run`, `Get` as usual. It also reads a file the `kavun` CLI compiled (which has no bindings).
Underneath are `(*vm.Bytecode).Encode` / `Decode`, with the same arguments (the envelope and the order of the
checks: `docs/vm.md` § Bytecode format).

**A loaded bytecode is trusted or refused — there is no verifier.** Decode does not inspect the code: bytecode
our compiler produced is valid by construction. Bytecode anybody else produced or changed is not held to anything:
it can import any registered module whatever the VM's grant, and malformed code can stop the run (Go stays
memory-safe and `Run` contains the failure as a fatal `internal` error — the host survives, the script does not).
There are deliberately no partial checks: a verifier that catches some forgeries would only make unsigned bytecode
look safer than it is. So:

- **Take source, not bytecode, from a party you do not trust**, and compile it under your own grant.
- **Set a key to know a blob is yours.** Without a signer, every blob is trusted as given. With one, `Encode`
  tags what it writes and `Decode` refuses a blob that is unsigned, changed by a single bit, or signed with
  another key (`vm.ErrUntrusted`). Nothing built into the engine can do this for you: the engine is open source,
  so a key compiled into it is public — the key must be your application's, and private to it.

```go
signer := vm.HMACSigner(key) // HMAC-SHA256; key: at least vm.MinHMACKeyLen (32) random bytes, kept secret
if err := compiled.Encode(w, signer); err != nil {
    return err
}

// ... later, possibly in another process of the same application
loaded, err := kavun.DecodeCompiled(r, signer)
if errors.Is(err, vm.ErrUntrusted) {
    return err // not ours, or changed since
}
if err != nil {
    return err // vm.ErrIncompatible: recompile from source; vm.ErrNotBytecode: not a compiled script
}
loaded.MustSet("x", core.IntValue(50))
err = loaded.Run(machine)
```

### Where the key lives

The signature answers one question — *did this application compile this blob?* — so it is only as good as the
key's secrecy. HMAC signs and verifies with the same key: whoever holds it can produce code your application will
run under its grant. Pick the shape by where code is compiled and where it runs:

| shape | signer | why |
| --- | --- | --- |
| compile and run in one process, nothing stored | none | the blob never leaves memory: it is your compiler's output by construction |
| a service compiles, stores, and runs later | HMAC, key only in the service | storage is outside the trust boundary; the key is not |
| built in one place, run in many (shipped apps) | asymmetric, private key at build only | a key inside a shipped binary can be extracted |
| development and tests (`kavun -key`) | HMAC, dev key | exercises the signed path; never a production key |

**The service is the only compiler — the production shape.** Clients and authors send source; the service compiles
it under its own grant, signs it, and stores it. Between compile and run the bytecode sits in a database, a cache or
object storage, or moves between replicas — places other people and systems can write to. The signature makes write
access to them worth nothing: a changed row is refused, not run. The key lives in the service's secret store (KMS,
vault), is loaded at startup, and never reaches a client, a repository or a developer machine.

```go
// saveRule compiles an author's source under the service's grant, signs it, and stores both.
func saveRule(ctx context.Context, db *sql.DB, signer vm.Signer, id string, src []byte) error {
    compiled, err := kavun.NewScript(src, "input", "decision").Compile()
    if err != nil {
        return err // the author's mistake: report it, store nothing
    }
    var code bytes.Buffer
    if err := compiled.Encode(&code, signer); err != nil {
        return err
    }
    _, err = db.ExecContext(ctx, `UPDATE rules SET source = $2, bytecode = $3 WHERE id = $1`, id, src, code.Bytes())
    return err
}

// loadRule answers a stored rule ready to run, or refuses one that was changed outside the service.
func loadRule(ctx context.Context, db *sql.DB, signer vm.Signer, id string) (*kavun.Compiled, error) {
    var src, code []byte
    err := db.QueryRowContext(ctx, `SELECT source, bytecode FROM rules WHERE id = $1`, id).Scan(&src, &code)
    if err != nil {
        return nil, err
    }
    compiled, err := kavun.DecodeCompiled(bytes.NewReader(code), signer)
    switch {
    case errors.Is(err, vm.ErrUntrusted):
        return nil, err // not compiled by this service, or changed since: alert, never run it
    case errors.Is(err, vm.ErrIncompatible):
        return kavun.NewScript(src, "input", "decision").Compile() // the engine or host moved on: recompile (and re-save)
    }
    return compiled, err
}
```

Recompiling holds the stored source to your grant, but the source is only as trustworthy as the database it came
from; if that matters, authenticate it too (an HMAC over it with the same key), or keep it where only the service
writes.

**Code built in one place and run in many.** When the runners are outside your control — a desktop or mobile app,
an on-premise install — any key they hold can be extracted, so a shared HMAC key proves nothing there. Bring an
asymmetric `vm.Signer`: the build holds the private key, the runners only the public one, so they can check that
code is yours but cannot produce any. Key custody, rotation and a key id (it can travel inside the tag) are the
signer's business:

```go
type ed25519Signer struct {
    priv ed25519.PrivateKey // nil where code only runs
    pub  ed25519.PublicKey
}

func (s ed25519Signer) Sign(data []byte) ([]byte, error) {
    if s.priv == nil {
        return nil, errors.New("this process cannot sign bytecode")
    }
    return ed25519.Sign(s.priv, data), nil
}

func (s ed25519Signer) Verify(data, tag []byte) error {
    if !ed25519.Verify(s.pub, data, tag) {
        return errors.New("signature mismatch")
    }
    return nil
}
```

### Compatibility

**Versions tell whether a trusted blob still means the same thing.** Every blob carries two: the engine's
(`vm.CoreVersion()`) and yours (`core.SetHostVersion(major, minor)`). `Decode` runs a blob iff, for each, the
major is equal and the blob's minor is not above the running one; otherwise `vm.ErrIncompatible` — recompile from
source. Your version is **mandatory once you register types or modules** (`Encode` and `Decode` refuse with
`vm.ErrHostVersionUnset` until it is set), and keeping it right is your duty — a stale version is undetectable:

- bump the **minor** when you add — a type, a module, a function in a free slot, a wider default grant;
- bump the **major** when something that exists changes meaning — you move `myFunc` from slot 3 to slot 4, and
  put `otherFunc` in slot 3: old bytecode calling slot 3 would now call `otherFunc`;
- a behaviour fix bumps nothing.

A blob whose host version is `0.0` was produced with no extensions registered and references only the engine, so
every application accepts it.

## Handling errors

A run ends in one of three ways: `nil`, the context's error, or a `*kavun.RuntimeError`. Compile errors are a
separate type and come from `Script.Compile`, never from a run.

```go
type RuntimeError struct {
    Kind     string              // "division_by_zero", "conversion", "user", … or "" if the failure carried no kind
    Category errs.Category      // runtime (builtin/member/module), user (raise), requirement (require), system (VM)
    Fatal    bool                // bypassed every recover(); the VM was stopped
    Message  string              // human-readable detail, without the kind prefix
    Payload  core.Value          // the error value's payload — a string for runtime errors,
                                 // whatever the script passed for raise(x)
    Trace    []ast.SourceFilePos // active frames, innermost first
}
```

Branch on the struct, never on the text:

```go
if err := compiled.Run(machine); err != nil {
    var re *kavun.RuntimeError
    switch {
    case errors.As(err, &re):
        if re.Fatal {
            log.Printf("script stopped: %s: %s at %s", re.Kind, re.Message, re.Trace[0])
            break
        }
        // A recoverable error simply means the script did not handle it.
        log.Printf("script failed: %s: %s", re.Kind, re.Message)
    default:
        // context cancellation, or a host-side problem
        log.Printf("run failed: %v", err)
    }
}
```

`Error()` renders `Runtime Error: <kind>: <message>` plus one `\n\tat file:line:col` line per frame — for a script
function called with too few arguments, `Runtime Error: wrong_num_arguments: (call) expected 2 arguments, got 1
argument` (`Kind` `wrong_num_arguments`, `Message` the part after it). The sentinel
comparisons keep working through `Unwrap`:

```go
errors.Is(err, errs.ErrStackOverflow)   // true for a stack overflow
errs.AsError(err)                        // the underlying *errs.Error
errs.IsCritical(err)                     // same answer as re.Fatal
```

### `Payload` is the structured channel

A runtime failure's payload is its message string. A script's own `raise(x)` passes `x` through untouched, so a
script can hand the host a structured rejection rather than a sentence:

```go
raise({reason: "limit_exceeded", field: "amount", max: 10000})
```

```go
if errors.As(err, &re) && re.Kind == "user" {
    reason, _ := re.Payload.AccessIndex(core.NewStringValue("reason"))
    // …
}
```

Every kind a run can answer, with its severity and category, is listed in
[reference/types/error.md § Every error kind](reference/types/error.md#every-error-kind).

### Which failures are the caller's fault

`Kind` says what failed, not whose fault it was — and it cannot: `int(x)` does not know where `x` came from.
Three kinds carry that signal:

| `Kind` | read it as |
| --- | --- |
| `requirement` | the script's own input check rejected your data; `Payload` says which field and why |
| `conversion`, `json_decoding`, `io` | the data or the world did not cooperate |
| anything else | most likely a defect in the script |

### Recoverable vs. fatal

| | reaches the script's `recover()` | means |
| --- | --- | --- |
| recoverable | yes | the script or its data is wrong, or the world refused; the script chose not to handle it |
| fatal | no | the VM could not continue: stack overflow, an internal invariant, a panicking host hook, a host setup mistake |

A panicking hook — in a custom `ValueTypeDescr`, a host builtin, anywhere — never crosses `Run`. It is contained
and answered as a fatal `internal` error carrying the panic text and the script's stack trace. Treat one as a bug
report about the host code, not as a condition to handle. See `docs/extending_types.md`.

### A script can answer instead of failing

When the host wants a decision rather than an error, the script assigns its outcome to a bound global and handles
the failure itself with a top-level `defer` (see `docs/language.md` § "Deferred calls"). The run then returns
`nil` and the host reads the outcome back:

```go
// script:
//   decision = {ok: true}
//   defer func() {
//       e := recover()
//       if e != undefined { decision = {ok: false, why: e.value()} }
//   }()
//   …rules…

if err := compiled.Run(machine); err != nil { /* a defect, not a rejection */ }
decision, _ := compiled.Get("decision")
```

### What survives a failed run

The globals hold whatever the script assigned up to the point of failure, and `Compiled.Get`/`GetAll` read them
after an error just as after a success.

## Reusing a VM after a failure

`Compiled.Run` and `Compiled.RunContext` call `VM.Reset` before every run, so **the same VM is usable again after
every outcome**: success, a recoverable error, a fatal error, a stack overflow, a contained panic, and a cancelled
context. No `Clear()` is needed for correctness — see [Memory Management](#memory-management) for what it is for.

One run at a time, though: a `VM` must never be shared between goroutines (see [Concurrency](#concurrency)).

Cancellation is the one outcome that is not an error the script could have seen: `RunContext` answers `ctx.Err()`,
and the script's deferred calls — top-level ones included — do **not** run.

`VM.Abort` (which `RunContext` calls when the context ends) takes effect at the script's next loop iteration or
function call, and between the callbacks of a callback member (`map`, `for_each`, `reduce`, …) — a member over a
million elements stops within one element. Nothing after that point runs: no callback answers a result, no
statement after the member executes. A host builtin that calls a script function through `VM.Call` gets an error
back when the VM is aborted; it must return that error unchanged, like any other, so the run can stop. The abort
stays in force until the next `Reset`: a `VM.Call` made in between answers the same error.

## Concurrency

`Script`, `Compiled`, and `VM` are **not thread-safe**. For parallel execution:

1. Each goroutine must use its own `Compiled` (via `Clone`)
2. Each goroutine must use its own VM
3. Protect shared resources with explicit locking

Safe pattern for parallel runs:

```go
base, err := script.Compile()
if err != nil {
    panic(err)
}

// Each goroutine clones the compiled code
clone, err := base.Clone()
if err != nil {
    panic(err)
}

// Each goroutine has isolated runtime resources
machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)

if err := clone.Run(machine); err != nil {
    panic(err)
}
```

For cancellable execution, use `RunContext(ctx, machine)`.

## dec128 process-global settings

Kavun's `decimal` is [dec128](https://github.com/jokruger/dec128), which has five process-global settings
(`SetDefaultScale`, `SetArithmeticRounding`, `SetLossPolicy`, `SetTrimOutput`, `SetNullValue`). Kavun never
sets them, and scripts cannot read or change them; the documented behaviour of `decimal` assumes dec128's
defaults. They are process-wide, so a host that changes them — directly or through another package — changes
what `/`, `sqrt()`, `pow()`, `json.encode` and, past the 19-place ceiling, `+ - *` return inside every script.
That is the host's decision to make. The members that take an explicit `(scale, mode)` — `round`,
`div_round`, `mul_round` and the rest of the `*_round` family — never read them.

## Time zone data

A script names zones by their IANA name (`t.in_zone("Europe/Kyiv")`, `time({…, zone: "Europe/Kyiv"})`); `"UTC"`
is always available. The host's own zone is never visible to a script: `"Local"` and `""` are not zone names,
and `clock.now()` answers in UTC. A host that wants scripts to work in "its" zone passes the zone **name** in
as a value, which keeps the dependency visible.

Zone rules come from the host's IANA tzdata, which Kavun does not bundle. Go reads it in this order: the
`ZONEINFO` environment variable (a directory or an uncompressed `zoneinfo.zip`), the system copy
(`/usr/share/zoneinfo` on Unix), `$GOROOT/lib/time/zoneinfo.zip`, and finally the embedded copy if the program
imports `time/tzdata`. The `kavun` CLI does import it, so it works on minimal container images; a library does not
make that choice for its embedder — add `import _ "time/tzdata"` to your own `main` if your hosts may lack tzdata.

Zone rules change several times a year, so the **same script can answer a different wall clock — and so a
different date — on hosts with different tzdata versions**. UTC and fixed offsets are never affected. Two
consequences for a host:

- **Know which version you run.** In a container image the version is fixed when the image is built (the OS
  `tzdata` package; `cat /usr/share/zoneinfo/+VERSION` or the first line of `/usr/share/zoneinfo/tzdata.zi` shows
  it on most systems). A Go build's embedded copy is the version of that Go release.
- **To replay a past result exactly**, keep the `zoneinfo.zip` of each tzdata version you ran with, record the
  version label next to the stored result (the zip itself carries no version), and run the replay with
  `ZONEINFO=/path/to/zoneinfo-<version>.zip`. `ZONEINFO` is process-wide: replaying several versions side by side
  in one process is not supported.

**Sentinel instants.** Data from other systems often carries `0001-01-01` ("unset", .NET's `default(DateTime)`)
or `9999-12-31` ("no end"). Both are real instants inside `time`'s range (years 1…9999) and both are
**truthy** — only `time()`, the unix epoch, is falsy. If a sentinel means "missing", map it to `undefined` before
it reaches the script, so the script's `if t` and its maybe-missing defaults see what the data means.

## Memory Management

By default, VM reuse is lazy: stack and frame references are not fully cleared between runs. This improves performance
but keeps some references alive longer (until overwritten).

For more aggressive memory release when memory pressure is critical:

```go
if err := compiled.Run(machine); err != nil {
    panic(err)
}

// Optional: explicitly release remaining stack/frame references
machine.Clear()
```

Use `Clear()` when you prioritize releasing memory over peak throughput.

## Advanced Patterns

### One-Shot Execution

If you prefer a simpler one-shot flow without explicit resource management:

```go
func RunOnce(src []byte) error {
    script := kavun.NewScript(src)
    machine := vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)
    compiled, err := script.Compile()
    if err != nil {
        return err
    }
    return compiled.Run(machine)
}
```

This pattern is simpler but loses the benefits of reusing compiled code and VM state across multiple executions.
