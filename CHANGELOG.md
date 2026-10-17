# Changelog

All notable changes per release. Versions follow [semver](https://semver.org).

## v0.1.0, 2026-10-17

- Added `deribit`, a REST client generated with oapi-codegen from [Deribit API version 2.1.1, OpenAPI 3.0.0](https://docs.deribit.com/specifications/deribit_openapi.json). It covers all 186 paths: market data, instruments, accounts, positions, orders, transfers and account administration.
- Added `deribitws` in `ws/`, generated with gen-asyncapi from [Deribit Subscriptions version 1.0.0, AsyncAPI 3.0.0](https://docs.deribit.com/specifications/deribit_asyncapi.json), covering all 43 subscription channels.
- Number fields use `float64`. `make generate` downloads both specs, applies the documented patches and regenerates the clients. The specs are not committed.
- Added the `go-deribit` agent skill with setup and usage references.
