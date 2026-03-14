package gs_test

import (
	"context"
	"fmt"

	"github.com/jokruger/gs"
	"github.com/jokruger/gs/value"
)

func Example() {
	src := `
each := func(seq, fn) {
    for x in seq { fn(x) }
}

sum := 0
mul := 1
each([a, b, c, d], func(x) {
	sum += x
	mul *= x
})`

	// create a new Script instance
	script := gs.NewScript([]byte(src))

	// set values
	script.Add("a", &value.Int{Value: 1})
	script.Add("b", &value.Int{Value: 9})
	script.Add("c", &value.Int{Value: 8})
	script.Add("d", &value.Int{Value: 4})

	// run the script
	compiled, err := script.RunContext(context.Background())
	if err != nil {
		panic(err)
	}

	// retrieve values
	sum := compiled.Get("sum")
	mul := compiled.Get("mul")
	fmt.Println(sum, mul)

	// Output:
	// 22 288
}
