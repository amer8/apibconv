#!/bin/bash
# scripts/gen_expected.sh
set -e

# Build the tool first
make build

BIN="./apibconv"
TD="test/integration/testdata"
EXP="$TD/expected"

mkdir -p "$EXP"

echo "Generating expected files..."

# Helper function to run conversion
# Usage: conv <input> <output_name> <format> <version_flag> <version_val> <protocol_flag> <protocol_val>
conv() {
    local input=$1
    local output=$2
    local format=$3
    local v_flag=$4
    local v_val=$5
    local p_flag=$6
    local p_val=$7

    echo "  $output"
    local cmd=("$BIN" -o "$EXP/$output" "--to" "$format")
    if [ -n "$v_flag" ]; then
        cmd+=("$v_flag" "$v_val")
    fi
    if [ -n "$p_flag" ]; then
        cmd+=("$p_flag" "$p_val")
    fi
    cmd+=("$TD/$input")

    "${cmd[@]}"
}

# Group 1: API Blueprint input
conv apiblueprint.apib expected_apiblueprint_to_asyncapi_v2_ws.yaml asyncapi "--asyncapi-version" "2.6" "--protocol" "ws"
conv apiblueprint.apib expected_apiblueprint_to_asyncapi_v2.json asyncapi "--asyncapi-version" "2.6" "--protocol" "kafka"
conv apiblueprint.apib expected_apiblueprint_to_asyncapi_v2.yaml asyncapi "--asyncapi-version" "2.6" "--protocol" "kafka"
conv apiblueprint.apib expected_apiblueprint_to_asyncapi_v3_wss.yaml asyncapi "--asyncapi-version" "3.0" "--protocol" "wss"
conv apiblueprint.apib expected_apiblueprint_to_asyncapi_v3.json asyncapi "--asyncapi-version" "3.0" "--protocol" "kafka"
conv apiblueprint.apib expected_apiblueprint_to_asyncapi_v3.yaml asyncapi "--asyncapi-version" "3.0" "--protocol" "kafka"
conv apiblueprint.apib expected_apiblueprint_to_openapi_v2.json openapi "--openapi-version" "2.0"
conv apiblueprint.apib expected_apiblueprint_to_openapi_v2.yaml openapi "--openapi-version" "2.0"
conv apiblueprint.apib expected_apiblueprint_to_openapi_v3_0.json openapi "--openapi-version" "3.0"
conv apiblueprint.apib expected_apiblueprint_to_openapi_v3_0.yaml openapi "--openapi-version" "3.0"
conv apiblueprint.apib expected_apiblueprint_to_openapi_v3_1.json openapi "--openapi-version" "3.1"
conv apiblueprint.apib expected_apiblueprint_to_openapi_v3_1.yaml openapi "--openapi-version" "3.1"

# Group 2: AsyncAPI v2 input (using .yaml as default source)
conv asyncapi_v2.yaml expected_asyncapi_v2_to_apiblueprint.apib apiblueprint
conv asyncapi_v2.yaml expected_asyncapi_v2_to_asyncapi_v3.json asyncapi "--asyncapi-version" "3.0" "--protocol" "kafka"
conv asyncapi_v2.yaml expected_asyncapi_v2_to_asyncapi_v3.yaml asyncapi "--asyncapi-version" "3.0" "--protocol" "kafka"
conv asyncapi_v2.yaml expected_asyncapi_v2_to_openapi_v2.json openapi "--openapi-version" "2.0"
conv asyncapi_v2.yaml expected_asyncapi_v2_to_openapi_v2.yaml openapi "--openapi-version" "2.0"
conv asyncapi_v2.yaml expected_asyncapi_v2_to_openapi_v3_0.json openapi "--openapi-version" "3.0"
conv asyncapi_v2.yaml expected_asyncapi_v2_to_openapi_v3_0.yaml openapi "--openapi-version" "3.0"
conv asyncapi_v2.yaml expected_asyncapi_v2_to_openapi_v3_1.json openapi "--openapi-version" "3.1"
conv asyncapi_v2.yaml expected_asyncapi_v2_to_openapi_v3_1.yaml openapi "--openapi-version" "3.1"

# Group 3: AsyncAPI v3 input
conv asyncapi_v3.yaml expected_asyncapi_v3_to_apiblueprint.apib apiblueprint
conv asyncapi_v3.yaml expected_asyncapi_v3_to_openapi_v2.json openapi "--openapi-version" "2.0"
conv asyncapi_v3.yaml expected_asyncapi_v3_to_openapi_v2.yaml openapi "--openapi-version" "2.0"
conv asyncapi_v3.yaml expected_asyncapi_v3_to_openapi_v3_0.json openapi "--openapi-version" "3.0"
conv asyncapi_v3.yaml expected_asyncapi_v3_to_openapi_v3_0.yaml openapi "--openapi-version" "3.0"
conv asyncapi_v3.yaml expected_asyncapi_v3_to_openapi_v3_1.json openapi "--openapi-version" "3.1"
conv asyncapi_v3.yaml expected_asyncapi_v3_to_openapi_v3_1.yaml openapi "--openapi-version" "3.1"

# Group 4: OpenAPI v2 JSON input
conv openapi_v2.json expected_openapi_v2_json_to_apiblueprint.apib apiblueprint
conv openapi_v2.json expected_openapi_v2_json_to_asyncapi_v2_amqp.yaml asyncapi "--asyncapi-version" "2.6" "--protocol" "amqp"
conv openapi_v2.json expected_openapi_v2_json_to_asyncapi_v2.json asyncapi "--asyncapi-version" "2.6" "--protocol" "kafka"
conv openapi_v2.json expected_openapi_v2_json_to_asyncapi_v2.yaml asyncapi "--asyncapi-version" "2.6" "--protocol" "kafka"
conv openapi_v2.json expected_openapi_v2_json_to_asyncapi_v3.json asyncapi "--asyncapi-version" "3.0" "--protocol" "kafka"
conv openapi_v2.json expected_openapi_v2_json_to_asyncapi_v3.yaml asyncapi "--asyncapi-version" "3.0" "--protocol" "kafka"
conv openapi_v2.json expected_openapi_v2_json_to_openapi_v3_0.json openapi "--openapi-version" "3.0"
conv openapi_v2.json expected_openapi_v2_json_to_openapi_v3_0.yaml openapi "--openapi-version" "3.0"
conv openapi_v2.json expected_openapi_v2_json_to_openapi_v3_1.json openapi "--openapi-version" "3.1"
conv openapi_v2.json expected_openapi_v2_json_to_openapi_v3_1.yaml openapi "--openapi-version" "3.1"

# Group 5: OpenAPI v2 YAML input
conv openapi_v2.yaml expected_openapi_v2_yaml_to_apiblueprint.apib apiblueprint
conv openapi_v2.yaml expected_openapi_v2_yaml_to_asyncapi_v2.json asyncapi "--asyncapi-version" "2.6" "--protocol" "kafka"
conv openapi_v2.yaml expected_openapi_v2_yaml_to_asyncapi_v2.yaml asyncapi "--asyncapi-version" "2.6" "--protocol" "kafka"
conv openapi_v2.yaml expected_openapi_v2_yaml_to_asyncapi_v3.json asyncapi "--asyncapi-version" "3.0" "--protocol" "kafka"
conv openapi_v2.yaml expected_openapi_v2_yaml_to_asyncapi_v3.yaml asyncapi "--asyncapi-version" "3.0" "--protocol" "kafka"
conv openapi_v2.yaml expected_openapi_v2_yaml_to_openapi_v3_0.json openapi "--openapi-version" "3.0"
conv openapi_v2.yaml expected_openapi_v2_yaml_to_openapi_v3_0.yaml openapi "--openapi-version" "3.0"
conv openapi_v2.yaml expected_openapi_v2_yaml_to_openapi_v3_1.json openapi "--openapi-version" "3.1"
conv openapi_v2.yaml expected_openapi_v2_yaml_to_openapi_v3_1.yaml openapi "--openapi-version" "3.1"

# Group 6: OpenAPI v3.0 JSON input
conv openapi_v3_0.json expected_openapi_v3_0_json_to_apiblueprint.apib apiblueprint
conv openapi_v3_0.json expected_openapi_v3_0_json_to_asyncapi_v2_http.yaml asyncapi "--asyncapi-version" "2.6" "--protocol" "http"
conv openapi_v3_0.json expected_openapi_v3_0_json_to_asyncapi_v2.json asyncapi "--asyncapi-version" "2.6" "--protocol" "kafka"
conv openapi_v3_0.json expected_openapi_v3_0_json_to_asyncapi_v2.yaml asyncapi "--asyncapi-version" "2.6" "--protocol" "kafka"
conv openapi_v3_0.json expected_openapi_v3_0_json_to_asyncapi_v3_http.yaml asyncapi "--asyncapi-version" "3.0" "--protocol" "http"
conv openapi_v3_0.json expected_openapi_v3_0_json_to_asyncapi_v3.json asyncapi "--asyncapi-version" "3.0" "--protocol" "kafka"
conv openapi_v3_0.json expected_openapi_v3_0_json_to_asyncapi_v3.yaml asyncapi "--asyncapi-version" "3.0" "--protocol" "kafka"
conv openapi_v3_0.json expected_openapi_v3_0_json_to_openapi_v3_1.json openapi "--openapi-version" "3.1"
conv openapi_v3_0.json expected_openapi_v3_0_json_to_openapi_v3_1.yaml openapi "--openapi-version" "3.1"

# Group 7: OpenAPI v3.0 YAML input
conv openapi_v3_0.yaml expected_openapi_v3_0_yaml_to_apiblueprint.apib apiblueprint
conv openapi_v3_0.yaml expected_openapi_v3_0_yaml_to_asyncapi_v2.json asyncapi "--asyncapi-version" "2.6" "--protocol" "kafka"
conv openapi_v3_0.yaml expected_openapi_v3_0_yaml_to_asyncapi_v2.yaml asyncapi "--asyncapi-version" "2.6" "--protocol" "kafka"
conv openapi_v3_0.yaml expected_openapi_v3_0_yaml_to_asyncapi_v3_mqtt.yaml asyncapi "--asyncapi-version" "3.0" "--protocol" "mqtt"
conv openapi_v3_0.yaml expected_openapi_v3_0_yaml_to_asyncapi_v3.json asyncapi "--asyncapi-version" "3.0" "--protocol" "kafka"
conv openapi_v3_0.yaml expected_openapi_v3_0_yaml_to_asyncapi_v3.yaml asyncapi "--asyncapi-version" "3.0" "--protocol" "kafka"
conv openapi_v3_0.yaml expected_openapi_v3_0_yaml_to_openapi_v3_1.json openapi "--openapi-version" "3.1"
conv openapi_v3_0.yaml expected_openapi_v3_0_yaml_to_openapi_v3_1.yaml openapi "--openapi-version" "3.1"

# Group 8: OpenAPI v3.1 JSON input
conv openapi_v3_1.json expected_openapi_v3_1_json_to_apiblueprint.apib apiblueprint
conv openapi_v3_1.json expected_openapi_v3_1_json_to_asyncapi_v2.json asyncapi "--asyncapi-version" "2.6" "--protocol" "kafka"
conv openapi_v3_1.json expected_openapi_v3_1_json_to_asyncapi_v2.yaml asyncapi "--asyncapi-version" "2.6" "--protocol" "kafka"
conv openapi_v3_1.json expected_openapi_v3_1_json_to_asyncapi_v3.json asyncapi "--asyncapi-version" "3.0" "--protocol" "kafka"
conv openapi_v3_1.json expected_openapi_v3_1_json_to_asyncapi_v3.yaml asyncapi "--asyncapi-version" "3.0" "--protocol" "kafka"

# Group 9: OpenAPI v3.1 YAML input
conv openapi_v3_1.yaml expected_openapi_v3_1_yaml_to_apiblueprint.apib apiblueprint
conv openapi_v3_1.yaml expected_openapi_v3_1_yaml_to_asyncapi_v2.json asyncapi "--asyncapi-version" "2.6" "--protocol" "kafka"
conv openapi_v3_1.yaml expected_openapi_v3_1_yaml_to_asyncapi_v2.yaml asyncapi "--asyncapi-version" "2.6" "--protocol" "kafka"
conv openapi_v3_1.yaml expected_openapi_v3_1_yaml_to_asyncapi_v3.json asyncapi "--asyncapi-version" "3.0" "--protocol" "kafka"
conv openapi_v3_1.yaml expected_openapi_v3_1_yaml_to_asyncapi_v3.yaml asyncapi "--asyncapi-version" "3.0" "--protocol" "kafka"

echo "Done."
