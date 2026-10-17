# go-deribit

[![Go Reference](https://pkg.go.dev/badge/github.com/psyb0t/go-deribit.svg)](https://pkg.go.dev/github.com/psyb0t/go-deribit)
[![CI](https://github.com/psyb0t/go-deribit/actions/workflows/pipeline.yml/badge.svg?branch=main)](https://github.com/psyb0t/go-deribit/actions/workflows/pipeline.yml)
[![version](https://raw.githubusercontent.com/psyb0t/go-deribit/badges/version.svg)](https://github.com/psyb0t/go-deribit/tags)
[![license](https://raw.githubusercontent.com/psyb0t/go-deribit/badges/license.svg)](LICENSE)
[![imported by](https://raw.githubusercontent.com/psyb0t/go-deribit/badges/importers.svg)](https://github.com/psyb0t/go-deribit/blob/badges/importers.md)

Unofficial Go client for [Deribit](https://docs.deribit.com), generated from the OpenAPI and AsyncAPI documents Deribit publishes. I am not affiliated with Deribit.

The root package `deribit` covers REST. Package `deribitws` in `ws/` covers JSON-RPC WebSocket subscriptions. REST uses [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen); WebSocket code uses [gen-asyncapi](https://github.com/psyb0t/gen-asyncapi). Generated files must be regenerated rather than edited by hand.

## Install

```sh
go get github.com/psyb0t/go-deribit
```

## Usage

Every REST endpoint in the spec has a typed method on `ClientWithResponses`. Responses decode into `JSON200` and the other status fields.

```go
package main

import (
	"context"
	"net/http"
	"os"

	"github.com/psyb0t/go-deribit"
)

func main() {
	auth := deribit.WithRequestEditorFn(
		func(_ context.Context, req *http.Request) error {
			req.Header.Set(
				"Authorization",
				"Bearer "+os.Getenv("DERIBIT_ACCESS_TOKEN"),
			)

			return nil
		},
	)

	client, err := deribit.NewClientWithResponses(
		"https://test.deribit.com/api/v2",
		auth,
	)
	if err != nil {
		panic(err)
	}

	res, err := client.PublicGetTimeWithResponse(context.Background())
	if err != nil {
		panic(err)
	}

	_ = res.JSON200
}
```

The same code is in `example_test.go`, so `make test` checks that it compiles. Server time is public and needs no token; the request editor shows how to attach one for private endpoints.

Method names come from the complete endpoint paths: `/public/get_time` becomes `PublicGetTimeWithResponse`.

The full API is in the [Go reference](https://pkg.go.dev/github.com/psyb0t/go-deribit); after `go get`, `go doc github.com/psyb0t/go-deribit` prints the REST package and `go doc github.com/psyb0t/go-deribit/ws` the WebSocket package locally.

## Auth

Private REST requests use `Authorization: Bearer <access-token>`. Read the token from `DERIBIT_ACCESS_TOKEN`, as the request editor above does. The `/public/auth` endpoint obtains an OAuth access token with client credentials, a client signature or a refresh token. This client does not refresh tokens automatically. The published REST spec omits security schemes. Never log authentication request URLs, access tokens or client secrets.

Use credentials for the selected environment:

| Environment | REST base URL | WebSocket URL |
|---|---|---|
| Testnet | `https://test.deribit.com/api/v2` | `wss://test.deribit.com/ws/api/v2` |
| Production | `https://www.deribit.com/api/v2` | `wss://deribit.com/ws/api/v2` |

## WebSocket

Import `github.com/psyb0t/go-deribit/ws` as `deribitws`. The generated client sends subscription requests and decodes notifications for all 43 channels, including platform state, announcements, order books, tickers, trades and private account updates. JSON-RPC notifications use `method: "subscription"`; `params.channel` selects the channel and `params.data` holds its payload. Private subscriptions and raw market-data channels require authentication on the same connection. The source document covers subscriptions only, so the caller handles authentication and command replies. See the [setup reference](.agents/skills/go-deribit/references/setup.md).

## Regenerating

The specs are not committed. `make generate` downloads both into the gitignored `api/` directory and regenerates both packages:

```sh
make generate
```

Each package's `gen.go` declares its steps:

- `scripts/fetch-spec.sh` downloads the [OpenAPI document](https://docs.deribit.com/specifications/deribit_openapi.json) and [AsyncAPI document](https://docs.deribit.com/specifications/deribit_asyncapi.json).
- `scripts/patch-spec.sh` removes schema-free REST GET bodies containing JSON-RPC examples and assigns operation IDs from complete paths. It repairs array and nullable schemas using the published response examples and descriptions. oapi-codegen uses `config.yaml`, which maps numbers to `float64`.
- `scripts/patch-asyncapi.sh` restores JSON-RPC envelopes, subscription request schemas and client operation references. It converts parameterized channel addresses to braces, matches notifications by channel, distinguishes raw order notifications, selects private subscription methods, fixes quote required-field typos and nullable numeric fields, and maps explicitly formatted floats to double precision. It also reconciles array, boolean, string and nullable schemas with the published notification examples, preserves documented optional fields and permits sparse incremental ticker statistics. All 43 channels and 86 operations remain.

Both patches normalize provider descriptions for generated comments without changing wire values.

`.github/workflows/regen.yml` does the same every day in CI. When the new spec changes the client, it runs `make lint`, `make test` and `make sec`. If a check fails, Claude edits the code and the checks run again, up to three times, and a review step reads the result. A passing change is committed with a CHANGELOG entry and tagged as the next release: a patch version for fixes and additions, a minor version for breaking changes.

## Agent integrations

The [skill](.agents/skills/go-deribit) teaches a coding agent how to use go-deribit when it writes Go. It works in any agent that reads `.agents/skills/`, and installs natively in the clients below.

### Claude Code

```bash
claude plugin marketplace add psyb0t/agents
claude plugin install go-deribit@psyb0t
```

### Codex

```bash
codex plugin marketplace add psyb0t/agents
codex plugin add go-deribit@psyb0t
```

Codex also picks the skill up with no install in any checkout of this repo.

### OpenClaw

The skill is published to ClawHub on every release:

```bash
openclaw skills install @psyb0t/go-deribit
```

## Development

Every command runs in the `Dockerfile.dev` image. The host needs only Docker and Make.

```sh
make help      # list targets
make dep       # go mod tidy + vendor
make lint      # shellcheck, shfmt, go fix, golangci-lint
make test      # go test -race
make sec       # gitleaks, govulncheck, semgrep
make generate  # download the specs and regenerate
```

## License

[MIT](LICENSE). The generated code comes from Deribit's published API specifications. Neither document declares a license; the specifications are not redistributed here.
