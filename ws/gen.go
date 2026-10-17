// Package deribitws provides generated Deribit subscription messages and a
// transport-independent JSON-RPC WebSocket client.
package deribitws

//go:generate bash ../scripts/fetch-spec.sh asyncapi.json
//go:generate bash ../scripts/patch-asyncapi.sh
//go:generate go tool gen-asyncapi --config=config.yaml ../api/asyncapi.fixed.json
