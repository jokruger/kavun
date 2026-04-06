# Project

## Layout Notes

The project keeps most tests under `tests/unit` instead of co-locating every `_test.go` file with the production code.

This is a deliberate choice. GS has a fairly broad runtime surface, and keeping tests in a dedicated tree makes larger behavior-oriented test cases easier to read, organize, and maintain. In practice, many tests exercise language and VM semantics across package boundaries, so grouping them by scenario is often clearer than scattering them throughout the source tree.

This is not the most idiomatic Go layout, and that tradeoff is intentional: for this project, readability and manageability take priority over strict colocation.

When adding or changing behavior, please add or update the relevant tests in `tests/unit`.
