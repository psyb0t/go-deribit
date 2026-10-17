#!/bin/bash
# Fixes example-only GET bodies and missing operation IDs for REST generation.
set -euo pipefail
trap 'on_error "$?" "$LINENO"' ERR

LOG_FILE="${LOG_FILE:-/tmp/$(basename "$0" .sh).log}"
exec > >(tee -a "$LOG_FILE") 2>&1

on_error() {
	local exit_code="$1"
	local line="$2"

	printf '{"level":"ERROR","file":"%s","line":%d,"msg":"command failed","exit_code":%d}\n' \
		"${BASH_SOURCE[1]}" "$line" "$exit_code" >&2
}

log() {
	local level="$1"
	shift

	jq -nc \
		--arg time "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
		--arg level "$level" \
		--arg file "${BASH_SOURCE[1]##*/}" \
		--arg line "${BASH_LINENO[0]}" \
		--arg func "${FUNCNAME[1]:-main}" \
		--arg msg "$*" \
		'{time:$time,level:$level,file:$file,line:($line|tonumber),func:$func,msg:$msg}' >&2
}

# GET bodies contain JSON-RPC examples only. REST uses query parameters.
# Remove these schema-free bodies to generate bodyless REST calls.
# Missing operation IDs become PascalCase names derived from complete paths.
# Normalize provider descriptions for generated comments without changing data.
# shellcheck disable=SC2016 # jq variables are not shell expansions
jq '
  # Published examples show arrays where these schemas declare one object.
  .components.schemas.instrument.properties.tick_size_steps |= {type:"array",items:.}
  | .components.schemas.PrivatePositionMoveResponse.properties.result |= {type:"array",items:.}
  # Volatility history rows are [timestamp, volatility] numeric pairs.
  | .components.schemas.PublicGetHistoricalVolatilityResponse.properties.result.items = {
      type:"array", minItems:2, maxItems:2, items:{type:"number",format:"double"},
      description:"Timestamp in milliseconds followed by historical volatility."
    }
  # Preserve explicit nulls in published response examples. Inline referenced
  # scalars only at nullable log properties, keeping request schemas unchanged.
  | .components.schemas as $schemas
  | .components.schemas.transaction_log.properties |= with_entries(
      if (.key | IN("commission","instrument_name","interest_pl","order_id","position","price","trade_id")) then
        .value |= ((if has("$ref") then $schemas[.["$ref"] | split("/") | last] else . end) + {nullable:true})
      else . end
    )
  | .components.schemas.PrivateAccountResponse.properties.result.properties.referrer_id.nullable = true
  | .components.schemas.PrivateAccountSummariesResponse.properties.result.properties.referrer_id.nullable = true
  | .components.schemas.PublicGetVolatilityIndexDataResponse.properties.result.properties.continuation.nullable = true
  | .components.schemas.PrivateGetBlockRfqsResponse.properties.result.properties.continuation = ($schemas.continuation + {nullable:true})
  | .components.schemas.user_trade.properties.matching_id.nullable = true
  | .paths |= with_entries(
    .key as $path | .value |= with_entries(
      if .key == "get" then
        .value.operationId = ($path | [splits("[/_]") | select(length > 0) | (.[0:1] | ascii_upcase) + .[1:]] | join(""))
        | if ((.value.requestBody.content // {}) | all(.[]; has("schema") | not)) then del(.value.requestBody) else . end
      else . end
    )
  )

| walk(if type == "object" and (.description? | type) == "string" then
    .description |= (
      gsub("[—–]"; ",")
      | gsub("comprehensive "; "")
      | gsub("Why do we use"; "Why does Deribit use")
      | gsub("[Ww]e recommend"; "Deribit recommends")
      | gsub("we use"; "Deribit uses")
      | gsub("we are interested in"; "Deribit measures")
      | gsub("we decay"; "Deribit decays")
      | gsub("we show"; "Deribit shows")
      | gsub("our Theta"; "the displayed Theta")
      | gsub("Our system"; "The Deribit system")
    )
  else . end)
' api/openapi.json >api/openapi.fixed.json
log INFO "patched REST spec"
