// Package deribit is a Go client for the Deribit REST API,
// generated with oapi-codegen from the OpenAPI document Deribit publishes.
package deribit

//go:generate bash scripts/fetch-spec.sh openapi.json
//go:generate bash scripts/patch-spec.sh
//go:generate go tool oapi-codegen --config=config.yaml api/openapi.fixed.json
