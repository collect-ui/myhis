# Agent Runtime Regression Records

- `latest-summary.md`: 最新一次回归执行摘要
- `latest-dialogue-report.md`: 最新一次 Markdown 对话测试报告
- `YYYYMMDD-HHMMSS/summary.md`: 某次回归摘要
- `YYYYMMDD-HHMMSS/dialogue-report.md`: 某次回归的 Markdown 对话测试报告
- `YYYYMMDD-HHMMSS/local.jsonl`: 本地稳定用例 `go test -json` 原始记录
- `YYYYMMDD-HHMMSS/real.jsonl`: 真实链路用例 `go test -json` 原始记录
- `YYYYMMDD-HHMMSS/local.log`: 本地稳定用例精简结果
- `YYYYMMDD-HHMMSS/real.log`: 真实链路用例精简结果

执行方式：

```bash
bash test/run_agent_runtime_regression.sh
```
