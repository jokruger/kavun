package mymod_test

import (
	"errors"
	"testing"

	"github.com/jokruger/kavun"
	"github.com/jokruger/kavun/core/member"
	"github.com/jokruger/kavun/errs"
	"github.com/jokruger/kavun/vm"

	"example.com/kavunembed/gadget"
	_ "example.com/kavunembed/mymod"
)

// TestEmbedded runs a script over the generated code: the names bound, the type's table and the module registered.
func TestEmbedded(t *testing.T) {
	if member.Lookup("double") != gadget.MemberDouble || member.Lookup("label") != gadget.MemberLabel {
		t.Fatalf("names not bound: double %d, label %d", member.Lookup("double"), member.Lookup("label"))
	}
	src := `
m := import("mymod")
out = [g.double(), g.len(), m.double(4), m.label, g.double]
`
	s := kavun.NewScript([]byte(src), "g", "out")
	c, err := s.Compile()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Set("g", gadget.New(21)); err != nil {
		t.Fatal(err)
	}
	if err := c.Run(vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize)); err != nil {
		t.Fatal(err)
	}
	out, err := c.Get("out")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), `[42, 21, 8, "embedded", <bound-member:gadget.double/0>]`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

// TestDoubleRefusesNonInt: a non-int argument raises invalid_argument_type, never a silent zero.
func TestDoubleRefusesNonInt(t *testing.T) {
	c, err := kavun.NewScript([]byte(`import("mymod").double("x")`)).Compile()
	if err != nil {
		t.Fatal(err)
	}
	err = c.Run(vm.NewVM(vm.DefaultMaxFrames, vm.DefaultStackSize))
	var re *kavun.RuntimeError
	if !errors.As(err, &re) || re.Kind != errs.KindInvalidArgumentType {
		t.Fatalf("got %v, want an %s error", err, errs.KindInvalidArgumentType)
	}
}
