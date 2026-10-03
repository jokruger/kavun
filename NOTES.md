# NOTES

## Member-function dispatch stays a `switch name` on the method's string (2026-10-03)

**Question.** Every `ValueTypeDescr.MethodCall` resolves the member with `switch name` on a string — up to 57
cases per type, 212 distinct names across `core`. Member sets are static, so the compiler could hand the VM a
`uint16` ID (or a function pointer) instead. Is the string switch the inefficiency it looks like?

**Measured** (`go test -run ^$ -bench 'Dispatch|MemberCall' ./benchmarks`, i7-9750H, go1.27.1):

| dispatch step alone, 71 real labels | string switch | uint16 switch | table by ID |
| ----------------------------------- | ------------: | ------------: | ----------: |
| one name (a real call site)         |        4.0 ns |        3.2 ns |      3.8 ns |
| names weighted by usage             |       14.2 ns |       11.3 ns |     13.3 ns |
| uniform random                      |       15.4 ns |       11.8 ns |     13.8 ns |

| whole in-language call           | ns/call |
| -------------------------------- | ------: |
| `len(a)` — builtin, integer slot |    12.7 |
| `a.len()` — 50-label switch      |    13.5 |
| `s.has_prefix("ab")`             |     145 |

In a loop that does nothing but `a.len()`, the `switch name` / `case "len"` lines are 4.8 % of CPU (pprof); the
universal-member compare in `Value.MethodCall` another 1.4 %. That 6 % is the ceiling for any dispatch change.

**Why it is this cheap.** Go compiles a string switch as a jump table on `len(name)` followed by 1/2/4/8-byte
integer compares of the content. The random rows are branch misprediction, which every strategy pays equally.

**Decision.** Keep the string switch. An ID scheme would buy ≤ 1 ns per call (< 7 % of the cheapest member,
< 1 % of a real one).
