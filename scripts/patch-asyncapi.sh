#!/bin/bash
# Repairs subscription schemas to describe JSON-RPC WebSocket wire frames.
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

# Operation message references point in the wrong direction for a client.
# Notification schemas omit the JSON-RPC envelope shown in their examples.
# Parenthesized channel parameters become generator-supported braces.
# Channel patterns distinguish otherwise identical subscription envelopes.
# Private channels need private/subscribe rather than the public example method.
# Correct best_add required-field typos and honor documented null prices/change.
# Use double precision for number schemas explicitly formatted as float.
SPEC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/api"
readonly SPEC_DIR
# Normalize provider descriptions for generated comments without changing data.
# shellcheck disable=SC2016 # jq variables are not shell expansions
jq '
def escaped: gsub("(?<char>[.\\[\\]{}()+*?^$|\\\\])"; "\\" + .char);
def channel_pattern($key; $channel):
  "^" + ($key | split(".") | map(
    if startswith("(") and endswith(")") then
      .[1:-1] as $parameter
      | ($channel.parameters[$parameter].enum // []) as $values
      | if ($values|length)>0 then "(" + ($values | map(tostring | escaped) | join("|")) + ")" else "[^.]+" end
    else escaped end
  ) | join("\\.")) + "$";
# Several payload schemas describe one element where their published examples
# show an array, including nested trades, orders, legs, and makers. Preserve
# the element schema and restore the array at the same location. Published
# explicit null values also need nullable schemas.
def wire_shape($value):
  if $value == null then .nullable = true
  elif ($value|type) == "boolean" and .type? == "string" then
    .type = "boolean"
  elif ($value|type) == "string" and .type? == "object"
       and ((.properties // {})|length) == 0 then
    .type = "string" | del(.properties, .additionalProperties)
  elif ($value|type) == "array" then
    (if .type? == "object" then {type:"array",items:.} else . end)
    | if .type? == "array" and (.items|type) == "object" then
        .items |= reduce $value[] as $item (.; wire_shape($item))
      else . end
  elif ($value|type) == "object" and (.properties?|type) == "object" then
    .properties |= with_entries(
      .key as $key
      | if ($value|has($key)) then .value |= wire_shape($value[$key])
        else . end)
  else . end;

# The examples include optional fields absent from their closed schemas.
.channels |= with_entries(
  .key as $channel
  | if $channel == "deribit_price_statistics.(index_name)" then
      .value.messages.subscription_message.payload.properties.data.properties
        .high_volatility = {type:"boolean"}
    elif $channel == "user.portfolio.(currency)" then
      .value.messages.subscription_message.payload.properties.data.properties
        .spot_reserve = {type:"number"}
    elif ($channel|startswith("user.orders.")) then
      .value.messages.subscription_message.payload.properties.data.properties
        .max_show = {type:"number"}
    elif ($channel|startswith("user.changes.")) then
      # Settlement price is explicitly optional for spot positions.
      .value.messages.subscription_message.payload.properties.data.properties |= (
        .positions.items.required -= ["settlement_price"]
        | .orders.items.properties.max_show = {type:"number"}
      )
    elif ($channel == "block_trade_confirmations" or $channel == "block_trade_confirmations.(currency)") then
      # The description marks app_name optional, and both examples omit it.
      .value.messages.subscription_message.payload.properties.data.required -= ["app_name"]
    elif $channel == "incremental_ticker.(instrument_name)" then
      # The channel description says deltas contain only changed fields.
      # Keep required instrument_name/timestamp, make nested stats sparse.
      .value.messages.subscription_message.payload.properties.data.properties
        .stats.required = []
    else . end
  | .value.messages.subscription_message |= (
    . as $message
    | .payload |= reduce $message.examples[]?.payload as $example
        (.; wire_shape($example))
  )
)

| .channels |= with_entries(
  .key as $key | .value as $channel
  | channel_pattern($key; $channel) as $pattern
  | (if ($key|startswith("user.")) or (($key|startswith("block_")) and ($key|startswith("block_rfq.trades.")|not)) then "private/subscribe" else "public/subscribe" end) as $method
  | .value.address |= gsub("\\((?<name>[^)]+)\\)"; "{" + .name + "}")
  | .value.messages.subscribe_request.payload = {
      type:"object", required:["jsonrpc","id","method","params"],
      properties:{jsonrpc:{type:"string",const:"2.0"}, id:{type:"integer"},
        method:{type:"string",const:$method},params:{type:"object",required:["channels"],
          properties:{channels:{type:"array",minItems:1,items:{type:"string",pattern:$pattern}}}}}}
  | .value.messages.subscription_message.payload as $params
  | .value.messages.subscription_message.payload = {
      type:"object",required:["jsonrpc","method","params"],
      properties:{jsonrpc:{type:"string",const:"2.0"},method:{type:"string",const:"subscription"},
        params:($params | .properties.channel={type:"string",pattern:$pattern} | .required += ["channel"])}}
  # Raw orders have their own channel schema; keep their decoder unambiguous.
  | if ($key|test("^user\\.orders\\..*\\.\\(interval\\)$")) then
      .value.messages.subscription_message.payload.properties.params.properties.channel.pattern |= sub("raw\\|"; "")
    else . end
)
| .operations |= with_entries(
    .value.messages |= map(."$ref" |=
      if endswith("/subscription_message") then sub("/subscription_message$";"/subscribe_request")
      else sub("/subscribe_request$";"/subscription_message") end))
| walk(if type == "object" then
    if (.type? == "number" or .type? == "string") and ((.description? // "")|test("`null`|\\bnull\\b")) then .nullable=true else . end
    | if has("required") then .required |= map(
        if . == "best_add_amount" then "best_ask_amount"
        elif . == "best_add_price" then "best_ask_price" else . end) else . end
    | if .type? == "number" and .format? == "float" then .format="double" else . end
  else . end)

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
' "$SPEC_DIR/asyncapi.json" >"$SPEC_DIR/asyncapi.fixed.json"
log INFO "patched AsyncAPI spec"
