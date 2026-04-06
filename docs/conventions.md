# Coding Conventions

This document describes conventions and contracts that should be followed when contributing to the GS project.

## Variadic Arguments: Immutability Contract

Functions that accept variadic arguments (`...Value`) must **never mutate** the arguments slice or its elements. This is both a Go best practice and a critical requirement for performance in this VM.

To avoid allocations, the VM passes stack slices directly to callees. The full capacity of these slices extends to the end of the stack array. If a callee appends to `args`, it corrupts subsequent stack frames.

Functions should not have side effects on caller state beyond their explicit return values. Mutating arguments violates this principle.
