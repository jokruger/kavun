// A modified version of Go's JSON implementation.

// Copyright 2010, 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"github.com/jokruger/kavun/core"
)

// Encode returns the JSON encoding of the object.
func Encode(o core.Value) ([]byte, error) {
	return o.EncodeJSON()
}
