package gadget

import "github.com/jokruger/kavun/core/member"

// The embedder's member ids: the range from member.FirstUserDefined on is the embedder's.
const (
	MemberDouble member.ID = member.FirstUserDefined + iota
	MemberLabel
)

// The embedder's member names, bound by the generated kavun_gen.go. MemberLabel is exported, since package mymod
// names the constant label by it.
//
//kavun:names
var memberNames = map[member.ID]string{
	MemberDouble: "double",
	MemberLabel:  "label",
}
