package path

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/core/member"
)

// A path segment: a module-owned type, its descriptor and member bodies in one root file.
//
// ## Overview
//
// The page file is `types/path_segment.md`: the dot of the Kavun name becomes an underscore.
//
//kavun:type path.segment

// TypeSegment is the segment's descriptor; its table is generated next to the module's init.
var TypeSegment = core.ValueTypeDescr{
	Methods: segmentMethods,
}

// Answer the number of octets in the segment.
//
//kavun:member len pure () -> int
func segmentLen(_ core.VM, v core.Value, _ member.ID, args []core.Value) (core.Value, error) {
	return core.Undefined, nil
}
