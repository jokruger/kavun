package vm

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"

	"github.com/jokruger/kavun/ast"
	"github.com/jokruger/kavun/core"
)

// Encoded bytecode is an envelope around the gob stream of a Bytecode:
//
//	"KVN" | tag length (u16) | tag | core major (u16) | core minor | host major | host minor | gob(Bytecode)
//	                               |<---------------------- signed region ------------------------------>|
//
// Integers are little-endian. The tag is the Signer's output over the signed region (length 0: unsigned), so a
// changed version is a trust failure, not a compatibility one. See docs/vm.md § Bytecode format.
const BytecodeMagic = "KVN"

// MaxTagLen is the longest signature tag an envelope may carry.
const MaxTagLen = 1024

// The engine's bytecode version (Version's rules, core/host_version.go). BUMP IT WITH EVERY CHANGE TO WHAT BYTECODE
// MEANS. Minor: an addition to a frozen, append-only table — an opcode (core/bytecode/opcodes), a member id
// (core/member/members), a builtin-function slot, a type id (core/value), a static pool kind (core.Static). Major:
// a change of meaning — an opcode's operand layout or semantics, a moved or reused id or slot, a field of
// Bytecode/core.Static/core.CompiledFunction, the gob or envelope format. A behavior fix bumps nothing.
const (
	coreVersionMajor = 1
	coreVersionMinor = 0
)

// CoreVersion answers the engine's bytecode version: Decode runs bytecode whose core version it Reads.
func CoreVersion() core.Version {
	return core.Version{Major: coreVersionMajor, Minor: coreVersionMinor}
}

// The load refusals, matchable with errors.Is. Each comes wrapped with the detail (which version, which reason).
var (
	ErrNotBytecode      = errors.New("not Kavun bytecode")
	ErrUntrusted        = errors.New("untrusted bytecode")
	ErrIncompatible     = errors.New("incompatible bytecode")
	ErrHostVersionUnset = errors.New("host version unset")
)

// Bytecode is a compiled instructions and constants.
type Bytecode struct {
	FileSet      *ast.SourceFileSet
	MainFunction *core.CompiledFunction
	Static       core.Static
	Imports      []uint8  // the builtin modules (type ids) the code imports, ascending; checked against the grant at Reset
	NumGlobals   int      // the global slots the code uses
	Bindings     []string // the host's binding names (kavun.NewScript), in global-slot order: Bindings[i] is slot i
}

// Encode writes the bytecode envelope to w, tagged by signer (nil: unsigned). With embedder types or modules
// registered, the host version must be set (core.SetHostVersion).
func (b *Bytecode) Encode(w io.Writer, signer Signer) error {
	if err := checkHostIdentity(); err != nil {
		return err
	}
	signed := binary.LittleEndian.AppendUint16(nil, coreVersionMajor)
	signed = binary.LittleEndian.AppendUint16(signed, coreVersionMinor)
	host := core.HostVersion()
	signed = binary.LittleEndian.AppendUint16(signed, host.Major)
	signed = binary.LittleEndian.AppendUint16(signed, host.Minor)
	buf := bytes.NewBuffer(signed)
	if err := gob.NewEncoder(buf).Encode(*b); err != nil {
		return fmt.Errorf("failed to encode bytecode: %w", err)
	}
	signed = buf.Bytes()

	var tag []byte
	if signer != nil {
		var err error
		if tag, err = signer.Sign(signed); err != nil {
			return fmt.Errorf("failed to sign bytecode: %w", err)
		}
		if len(tag) == 0 || len(tag) > MaxTagLen {
			return fmt.Errorf("failed to sign bytecode: tag length %d (want 1–%d)", len(tag), MaxTagLen)
		}
	}

	out := append([]byte(BytecodeMagic), 0, 0)
	binary.LittleEndian.PutUint16(out[len(BytecodeMagic):], uint16(len(tag)))
	out = append(out, tag...)
	for _, part := range [][]byte{out, signed} {
		if _, err := w.Write(part); err != nil {
			return fmt.Errorf("failed to encode bytecode: %w", err)
		}
	}
	return nil
}

// Decode reads a bytecode envelope from r (to EOF) and decodes it into b. The checks run in this order, each
// refusing with its own sentinel error: trust (signer set: the tag must be present and verify; nil: the tag is
// ignored), host identity (ErrHostVersionUnset), compatibility (core version, then host version), then the gob
// decode. Nothing else is checked: trusted bytecode is our compiler's output, valid by construction.
//
// NB: files in b.FileSet.File does not have their 'set' field properly set to b.FileSet as it's private field and not
// serialized by gob encoder/decoder.
func (b *Bytecode) Decode(r io.Reader, signer Signer) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read bytecode: %w", err)
	}
	env, err := readEnvelope(data)
	if err != nil {
		return err
	}
	if signer != nil {
		if len(env.tag) == 0 {
			return fmt.Errorf("%w: unsigned, and this application requires a signature", ErrUntrusted)
		}
		if err := signer.Verify(env.signed, env.tag); err != nil {
			return fmt.Errorf("%w: signature does not verify: %w", ErrUntrusted, err)
		}
	}
	if err := checkHostIdentity(); err != nil {
		return err
	}
	if !CoreVersion().Reads(env.core) {
		return fmt.Errorf("%w: core version %s (this engine: %s); recompile from source", ErrIncompatible, env.core, CoreVersion())
	}
	if host := core.HostVersion(); env.host != (core.Version{}) && !host.Reads(env.host) {
		return fmt.Errorf("%w: host version %s (this application: %s); recompile from source", ErrIncompatible, env.host, host)
	}
	if err := gob.NewDecoder(bytes.NewReader(env.payload)).Decode(b); err != nil {
		return fmt.Errorf("%w: %w", ErrNotBytecode, err)
	}
	// derived data the hot loop relies on (the cached string rune counts); idempotent
	b.Static.BuildStringLens()
	return nil
}

// BytecodeInfo is what Inspect reads from an encoded bytecode without judging it.
type BytecodeInfo struct {
	Core, Host core.Version
	Signed     bool      // a tag is present
	Verify     error     // with a signer: nil when the tag verifies, else the refusal Decode would answer
	Bytecode   *Bytecode // the decoded payload (Imports, Bindings, …), or nil when it does not decode
	PayloadErr error     // why the payload did not decode
}

// Inspect reads an encoded bytecode's metadata for tooling (`kavun -info`): the versions, whether it is signed,
// and — with a signer — whether the tag verifies, then the payload. It never refuses on trust or compatibility;
// that is how one finds out why Decode refuses. The only error is ErrNotBytecode/ErrIncompatible for a stream that
// has no envelope at all. Nothing is run or registered.
func Inspect(r io.Reader, signer Signer) (*BytecodeInfo, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read bytecode: %w", err)
	}
	env, err := readEnvelope(data)
	if err != nil {
		return nil, err
	}
	info := &BytecodeInfo{Core: env.core, Host: env.host, Signed: len(env.tag) > 0}
	if signer != nil {
		if !info.Signed {
			info.Verify = fmt.Errorf("%w: unsigned", ErrUntrusted)
		} else if err := signer.Verify(env.signed, env.tag); err != nil {
			info.Verify = fmt.Errorf("%w: signature does not verify: %w", ErrUntrusted, err)
		}
	}
	b := &Bytecode{}
	if err := gob.NewDecoder(bytes.NewReader(env.payload)).Decode(b); err != nil {
		info.PayloadErr = err
	} else {
		info.Bytecode = b
	}
	return info, nil
}

// envelope is an encoded bytecode split at its fixed fields; the slices alias the input.
type envelope struct {
	tag     []byte
	signed  []byte // versions + payload: what the tag covers
	core    core.Version
	host    core.Version
	payload []byte // gob(Bytecode)
}

const versionsLen = 8

func readEnvelope(data []byte) (envelope, error) {
	m := len(BytecodeMagic)
	if len(data) < m || string(data[:m]) != BytecodeMagic {
		return envelope{}, fmt.Errorf("%w: missing header", ErrNotBytecode)
	}
	if len(data) >= m+3 && isDigit(data[m]) && isDigit(data[m+1]) && isDigit(data[m+2]) {
		// "KVN002" and older: a tag length cannot spell three digits, since 0x3030 > MaxTagLen
		return envelope{}, fmt.Errorf("%w: format %s predates versioned bytecode; recompile from source", ErrIncompatible, data[:m+3])
	}
	if len(data) < m+2 {
		return envelope{}, fmt.Errorf("%w: truncated header", ErrNotBytecode)
	}
	n := int(binary.LittleEndian.Uint16(data[m:]))
	if n > MaxTagLen {
		return envelope{}, fmt.Errorf("%w: tag length %d (max: %d)", ErrNotBytecode, n, MaxTagLen)
	}
	rest := data[m+2:]
	if len(rest) < n+versionsLen {
		return envelope{}, fmt.Errorf("%w: truncated header", ErrNotBytecode)
	}
	env := envelope{tag: rest[:n], signed: rest[n:]}
	v := env.signed
	env.core = core.Version{Major: binary.LittleEndian.Uint16(v[0:]), Minor: binary.LittleEndian.Uint16(v[2:])}
	env.host = core.Version{Major: binary.LittleEndian.Uint16(v[4:]), Minor: binary.LittleEndian.Uint16(v[6:])}
	env.payload = v[versionsLen:]
	return env, nil
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// checkHostIdentity refuses to encode or decode while embedder extensions are registered and the host version is
// unset: such bytecode could reference them, and without a version no other application could tell it apart.
func checkHostIdentity() error {
	if core.HasExtensions() && core.HostVersion() == (core.Version{}) {
		return fmt.Errorf("%w: embedder types or modules are registered; call core.SetHostVersion", ErrHostVersionUnset)
	}
	return nil
}

// MustFormatInstructions returns human readable string representations of compiled instructions.
func (b *Bytecode) MustFormatInstructions() []string {
	r, err := FormatInstructions(b.MainFunction.Instructions, 0)
	if err != nil {
		panic(fmt.Errorf("failed to format instructions: %w", err))
	}
	return r
}

// FormatInstructions returns human readable string representations of compiled instructions.
func (b *Bytecode) FormatInstructions() ([]string, error) {
	return FormatInstructions(b.MainFunction.Instructions, 0)
}

// MustFormatStatics returns human readable string representations of compiled static values.
func (b *Bytecode) MustFormatStatics() []string {
	r, err := b.FormatStatics()
	if err != nil {
		panic(fmt.Errorf("failed to format constants: %w", err))
	}
	return r
}

// FormatStatics returns human readable string representations of compiled static values.
func (b *Bytecode) FormatStatics() (output []string, err error) {
	for i, v := range b.Static.Primitives {
		output = append(output, fmt.Sprintf("[% 3d] %s (%s|%v)", i, v.Value().String(), v.Value().TypeName(), v))
	}

	for i, v := range b.Static.Decimals {
		output = append(output, fmt.Sprintf("[% 3d] %s (decimal)", i, v.String()))
	}

	for i, v := range b.Static.Strings {
		output = append(output, fmt.Sprintf("[% 3d] %s (string)", i, v))
	}

	for i, v := range b.Static.Runes {
		output = append(output, fmt.Sprintf("[% 3d] %s (runes)", i, string(v.Elements)))
	}

	for i, v := range b.Static.Bytes {
		output = append(output, fmt.Sprintf("[% 3d] %v (bytes)", i, v.Elements))
	}

	for i, v := range b.Static.Times {
		output = append(output, fmt.Sprintf("[% 3d] %s (time)", i, v.Format("2006-01-02T15:04:05.999999999Z07:00")))
	}

	for i, v := range b.Static.FormatSpecs {
		output = append(output, fmt.Sprintf("[% 3d] %s (format spec)", i, v.Text))
	}

	for i, v := range b.Static.CompiledFunctions {
		output = append(output, fmt.Sprintf("[% 3d] (compiled function)", i))
		t, err := FormatInstructions(v.Instructions, 0)
		if err != nil {
			return nil, err
		}
		for _, l := range t {
			output = append(output, fmt.Sprintf("     %s", l))
		}
		continue
	}

	for i, v := range b.Static.NameLists {
		output = append(output, fmt.Sprintf("[% 3d] %v (name list)", i, v))
	}

	for i, v := range b.Static.Ranges {
		output = append(output, fmt.Sprintf("[% 3d] range(%d, %d, %d) (range)", i, v.Start, v.Stop, v.Step))
	}

	return
}
