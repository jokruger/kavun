// Package modkit holds what every builtin module shares at its boundary: the one translation point from a Go
// error to a raise.
package modkit

import (
	"github.com/jokruger/kavun/core"
	"github.com/jokruger/kavun/errs"
)

// Done is the answer of a "statement-like" module function — one whose success has no result. Success answers
// undefined, the language's spelling of "no result"; failure raises. Nothing to check after the call.
func Done(context string, err error) (core.Value, error) {
	if err == nil {
		return core.Undefined, nil
	}
	return core.Undefined, errs.NewIOError(context, err)
}

// RaiseIO translates a Go error from the os/exec family into a recoverable "io" error. Call it at the point a
// value-returning module function has already failed.
func RaiseIO(context string, err error) (core.Value, error) {
	return core.Undefined, errs.NewIOError(context, err)
}

// RaiseGo translates a Go error from any other library (regexp, encoding/*, time) into a recoverable error of
// the kind that module answers with. This is the SINGLE translation point for that module: a Go error must never
// travel further into the runtime, because errs.IsCritical reads a non-*errs.Error as fatal.
func RaiseGo(kind string, context string, err error) (core.Value, error) {
	return core.Undefined, errs.NewRecoverableError(kind, "("+context+") "+err.Error())
}
