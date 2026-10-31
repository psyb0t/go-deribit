# Changelog

All notable changes per release. Versions follow [semver](https://semver.org).

## v0.2.1, 2026-10-31

- Documentation only. The doc comment on `ChartVolume` in the REST client now reads "Traded volume within a single candle" instead of "todo".
- The doc comments on the 43 subscription request methods in `ws/` link to docs.deribit.com/articles/notifications instead of the old deribit.mintlify.app address.
- No exported identifiers changed.

## v0.2.0, 2026-10-29

- Regenerated from Deribit's current OpenAPI and AsyncAPI specs. The AsyncAPI rewrite broke `scripts/patch-asyncapi.sh`, so scheduled regeneration had stopped.
- Breaking, REST: 122 model types lose the numeric suffix oapi-codegen added to avoid name collisions, for example `PrivateBuyAndSellResponse2` is now `PrivateBuyAndSellResponse`. `ErrorResponse`, `DepositResponse`, `CurrencyPortfolioCurrency`, `GetlistCustodyAccounts200responseResponse` and `Getunsubscribe200responseResponse` are removed. The 186 operations are unchanged.
- Breaking, WebSocket: the `user.changes` channels no longer have the singular `Position` field, only `Positions`. Nested arrays that were element types are now typed slices with `Item` types, such as the trades in the `block_rfq` channels and `trade_allocations`. Order `price` is a union type, and the quote `best_bid_amount` and `best_ask_amount` fields are pointers. The block trade confirmation `state` and `counterparty_state` fields, the `instrument.creation` payload and the `user.lsp` configuration are typed.
- `scripts/patch-asyncapi.sh` makes `settlement_price` optional and adds `max_show` on the `positions` and `orders` arrays the spec now defines, instead of building `positions` from the singular `position` schema.

## v0.1.4, 2026-10-28

- Updated Semgrep in the development image to 1.182.1 and refreshed its locked dependencies. certifi 2026.10.23 drops the emSign root CAs whose intermediate keys were compromised (GHSA-6q48-6c7w-9gw2).
- Dependabot no longer opens pull requests for `mcp` or the OpenTelemetry packages in `requirements-semgrep.txt`. Semgrep pins their versions itself, so bumping one of them on its own cannot install.
- Updated kin-openapi, which the code generator uses, from 0.142.0 to 0.149.0. The newer version fixes several panics on malformed specs reported in GitHub security advisories.
- The security scan merges its two gitleaks runs into one SARIF run. GitHub code scanning rejects a report with two runs from the same tool, so no scan results had been uploaded.

## v0.1.3, 2026-10-27

- The development image is based on Debian (`golang:1.26.10-trixie`) instead of Alpine. Alpine keeps only the latest build of each package, so the exact pins broke the image build whenever a package was updated. System packages now come from the snapshot.debian.org date the base image was built from, which keeps them fixed for a given base image digest. jq 1.8.2, shellcheck 0.11.0 and shfmt 3.13.1 keep their versions, and Semgrep runs in its own virtual environment.
- Raised the minimum Go version in `go.mod` from 1.26.9 to 1.26.10, which fixes the `crypto/x509` vulnerability GO-2026-6809 found by govulncheck.
- The issue-pull workflow runs at hours derived from the repository name instead of the same four hours as every other repository.

## v0.1.2, 2026-10-17

- `make sec` no longer fails on the generated client's free-form JSON fields. Semgrep's unsafe-deserialization-interface rule is skipped for `client.gen.go` only. It still runs on handwritten code and the WebSocket package, and every other rule still runs on the generated client.
- The v0.1.1 documentation update (the `go doc` lookup section, the license field and the OpenClaw `requires` fix in the agent skill) reaches ClawHub with this release, because the v0.1.1 pipeline stopped at that check.

## v0.1.1, 2026-10-17

- Documentation only. The agent skill explains how to look up every endpoint, parameter and model with `go doc` or on pkg.go.dev, and declares its license.
- The README points to the full API reference, and the examples in the README and the skill were checked against the current code.
- `AGENTS.md` says the AsyncAPI patch makes `app_name` optional on block trade confirmations, as the generated code does.

## v0.1.0, 2026-10-17

- Added `deribit`, a REST client generated with oapi-codegen from [Deribit API version 2.1.1, OpenAPI 3.0.0](https://docs.deribit.com/specifications/deribit_openapi.json). It covers all 186 paths: market data, instruments, accounts, positions, orders, transfers and account administration.
- Added `deribitws` in `ws/`, generated with gen-asyncapi from [Deribit Subscriptions version 1.0.0, AsyncAPI 3.0.0](https://docs.deribit.com/specifications/deribit_asyncapi.json), covering all 43 subscription channels.
- Number fields use `float64`. `make generate` downloads both specs, applies the documented patches and regenerates the clients. The specs are not committed.
- Added the `go-deribit` agent skill with setup and usage references.
