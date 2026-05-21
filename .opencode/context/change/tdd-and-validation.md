# TDD And Validation

Default sequence from `AGENTS.md`:

1. `make check`
2. `make tdd`
3. implement the smallest correct change
4. `make verify`
5. add the narrowest relevant `make smoke*`

Use narrower targets when the touched path is obvious:

- pure Go behavior -> `make test`, `make test-behavioral`, `make test-behavioral-roundtrip`
- topology or layers -> `make arch-guard`, `make drift-detect`
- runtime bindings or contracts -> direct `raccoon-cli check topology`, `check bindings`, or `check contracts`
- significant changes -> `make check-deep`
- codegen area -> `make codegen-check`, `make codegen-test`, `make codegen-integrated`, `make codegen-equivalence`, `make codegen-validate-all`
- `tools/raccoon-cli/` -> `make raccoon-test`

Rule:

- `make verify` is not runtime proof
- `make check-deep` is not a substitute for the relevant smoke
