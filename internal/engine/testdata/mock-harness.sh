#!/usr/bin/env bash
# Mock harness for testing sealed execution mode.
# Usage: mock-harness.sh <condition> <run_number> <system_prompt_path> <model>
#
# Writes a result.json to a temp directory and prints the path to stdout.

set -euo pipefail

CONDITION="$1"
RUN_NUMBER="$2"
PROMPT_PATH="$3"
MODEL="$4"

OUTPUT_DIR="${CHIRON_TEST_OUTPUT_DIR:-$(mktemp -d)}"
RESULT_FILE="${OUTPUT_DIR}/result.json"

cat > "${RESULT_FILE}" <<JSONEOF
{
  "type": "result",
  "subtype": "success",
  "result": "mock sealed response for ${CONDITION}",
  "num_turns": 3,
  "duration_ms": 4200,
  "total_cost_usd": 0.0,
  "usage": {
    "input_tokens": 800,
    "output_tokens": 350
  },
  "tool_calls_observed": ["read", "bash"],
  "tool_summary": {"read": 1, "bash": 2},
  "model": "${MODEL}",
  "executor": "mock-harness"
}
JSONEOF

echo "${RESULT_FILE}"
