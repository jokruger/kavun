package core

import (
	"fmt"
	"slices"
)

// Version is a bytecode compatibility version: major.minor. A minor bump is an addition to a frozen, append-only
// table (an opcode, a member id, a builtin slot, a type id, a static pool kind; for an embedder: a type, module,
// function or a wider grant policy); a major bump is a change of meaning of something that exists (a layout, a moved
// id or slot, a codec format). A behavior fix, or a new member under an existing id, bumps nothing.
type Version struct {
	Major, Minor uint16
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
}

// Reads reports whether an engine at version v runs bytecode produced at version file: the same major, and a file
// minor not above the engine's (an older engine cannot know a newer addition).
func (v Version) Reads(file Version) bool {
	return file.Major == v.Major && file.Minor <= v.Minor
}

var (
	hostVersion   Version
	embedderTypes [256]bool // type ids 192–255 registered through SetValueType (embedder types and modules)
)

// SetHostVersion sets the embedder's own compatibility version, which travels in every bytecode this process
// encodes and is checked on every decode beside the engine's (vm.CoreVersion). Call it at init, beside the
// registration of the embedder's types and modules, and bump it by Version's rules whenever they change. 0.0 means
// unset: legal only while no embedder type or module is registered (HasExtensions), and then the bytecode is
// readable by every application, since it can reference only what the engine itself provides.
func SetHostVersion(major, minor uint16) {
	hostVersion = Version{Major: major, Minor: minor}
}

// HostVersion answers the version set with SetHostVersion, 0.0 when unset.
func HostVersion() Version {
	return hostVersion
}

// HasExtensions reports whether an embedder type or module (type id 192–255) is registered in this process.
// Bytecode produced with extensions can reference them, so encoding and decoding it requires a host version.
func HasExtensions() bool {
	return slices.Contains(embedderTypes[:], true)
}
