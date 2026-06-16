#!/usr/bin/env bash
set -u

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RESULT_DIR="$ROOT_DIR/test/test-results/agent-runtime"
STAMP="$(date '+%Y%m%d-%H%M%S')"
RUN_DIR="$RESULT_DIR/$STAMP"

mkdir -p "$RUN_DIR"

LOCAL_PATTERN='TestAgentRuntimeSessionRunFlow$|TestAgentRuntimeSessionRunFlowWithoutCredentialFallsBackToMock$|TestResolveAgentCredentialEmptyWhenNoEnvAndNoAuth$|TestResolveAgentCredentialFromCodexAuth$|TestLoadAgentConversationInputPreservesOrderAndRoleMapping$'
REAL_PATTERN='TestAgentRuntimeSessionRunFlowRealOpenAI$|TestAgentRuntimeSessionMultiTurnRealChat$'

run_suite() {
  local suite_name="$1"
  local pattern="$2"
  local json_file="$RUN_DIR/${suite_name}.jsonl"
  local text_file="$RUN_DIR/${suite_name}.log"

  (
    cd "$ROOT_DIR" &&
    go test -json -run "$pattern" -v ./...
  ) >"$json_file" 2>&1
  local status=$?

  if command -v jq >/dev/null 2>&1; then
    jq -r 'select(.Action=="pass" or .Action=="fail" or .Action=="skip") | [.Time, .Package, .Test, .Action, (.Elapsed // "")] | @tsv' "$json_file" >"$text_file" || true
  else
    cp "$json_file" "$text_file"
  fi

  return $status
}

build_markdown_report() {
  local report_file="$RUN_DIR/dialogue-report.md"

  python3 - "$RUN_DIR/local.jsonl" "$RUN_DIR/real.jsonl" "$report_file" <<'PY'
import json
import sys
from collections import OrderedDict

local_json, real_json, report_file = sys.argv[1], sys.argv[2], sys.argv[3]

scenario_titles = OrderedDict([
    ("mock_fallback", "本地回退对话"),
    ("real_single_turn", "真实单轮对话"),
    ("real_multi_turn", "真实多轮对话"),
])

field_titles = {
    "system_prompt": "System Prompt",
    "user_input": "用户输入",
    "assistant_output": "助手输出",
    "turn_1_user_input": "第 1 轮用户输入",
    "turn_1_assistant_output": "第 1 轮助手输出",
    "turn_2_user_input": "第 2 轮用户输入",
    "turn_2_assistant_output": "第 2 轮助手输出",
    "result": "结果",
}

scenarios = OrderedDict((key, OrderedDict()) for key in scenario_titles)

for path in (local_json, real_json):
    with open(path, "r", encoding="utf-8") as fh:
        for raw in fh:
            raw = raw.strip()
            if not raw:
                continue
            try:
                item = json.loads(raw)
            except json.JSONDecodeError:
                continue
            output = item.get("Output", "")
            marker = "REPORT|"
            if marker not in output:
                continue
            payload = output.split(marker, 1)[1].strip()
            parts = payload.split("|", 2)
            if len(parts) != 3:
                continue
            scenario, key, value = parts
            if scenario not in scenarios:
                scenarios[scenario] = OrderedDict()
            scenarios[scenario][key] = value.replace("\\n", "\n")

lines = []
lines.append("# Agent Runtime 对话测试报告")
lines.append("")
lines.append(f"- 生成目录: {report_file.rsplit('/', 1)[0]}")
lines.append("")

for scenario, title in scenario_titles.items():
    data = scenarios.get(scenario, {})
    if not data:
        continue
    lines.append(f"## {title}")
    lines.append("")
    for key, label in field_titles.items():
        if key not in data:
            continue
        value = data[key]
        lines.append(f"- {label}:")
        for line in value.splitlines() or [""]:
            lines.append(f"  {line}")
    lines.append("")

with open(report_file, "w", encoding="utf-8") as fh:
    fh.write("\n".join(lines).rstrip() + "\n")
PY
}

LOCAL_STATUS=0
REAL_STATUS=0

run_suite "local" "$LOCAL_PATTERN" || LOCAL_STATUS=$?
run_suite "real" "$REAL_PATTERN" || REAL_STATUS=$?
build_markdown_report

{
  echo "# Agent Runtime Regression Report"
  echo
  echo "- Time: $(date '+%Y-%m-%d %H:%M:%S %Z')"
  echo "- Run directory: $RUN_DIR"
  echo "- Local suite exit code: $LOCAL_STATUS"
  echo "- Real suite exit code: $REAL_STATUS"
  echo
  echo "## Suite Files"
  echo
  echo "- local json: $RUN_DIR/local.jsonl"
  echo "- local log: $RUN_DIR/local.log"
  echo "- real json: $RUN_DIR/real.jsonl"
  echo "- real log: $RUN_DIR/real.log"
  echo "- dialogue report: $RUN_DIR/dialogue-report.md"
} >"$RUN_DIR/summary.md"

cp "$RUN_DIR/summary.md" "$RESULT_DIR/latest-summary.md"
cp "$RUN_DIR/dialogue-report.md" "$RESULT_DIR/latest-dialogue-report.md"

echo "$RUN_DIR"
exit 0
