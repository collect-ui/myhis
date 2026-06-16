#!/usr/bin/env node

const fs = require("fs");
const path = require("path");
const { spawnSync } = require("child_process");
const { chromium } = require("playwright");

const PAGE_URL = process.env.WEBSHELL_EDITOR_POOL_PAGE_URL || "http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool";
const API_URL = process.env.WEBSHELL_EDITOR_POOL_API_URL || "http://127.0.0.1:8015/template_data/data";
const OUT_DIR = process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR || "/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation";

const PY_PROJECT_CODE = "backend";
const PY_PROJECT_NAME = "月神后端";
const GO_PROJECT_CODE = "autodesk";
const GO_PROJECT_NAME = "后端客户端";

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function runCurl(service, data) {
  const payload = JSON.stringify(Object.assign({ service }, data || {}));
  const res = spawnSync("curl", [
    "--noproxy",
    "*",
    "-sS",
    "-m",
    "40",
    `${API_URL}?service=${service}`,
    "-H",
    "Content-Type: application/json",
    "--data",
    payload,
  ], { encoding: "utf8" });
  if (res.status !== 0) {
    throw new Error(res.stderr || `curl failed: ${service}`);
  }
  let out = {};
  try {
    out = JSON.parse(String(res.stdout || "{}"));
  } catch (error) {
    throw new Error(`parse response failed (${service}): ${error.message}`);
  }
  if (!out || String(out.code || "") !== "0" || out.success === false) {
    throw new Error(`${service} failed: ${out?.msg || "unknown error"}`);
  }
  return out;
}

function normalizePath(input) {
  return String(input || "").replace(/\\/g, "/").replace(/\/+$/g, "");
}

function escapeRegExp(input) {
  return String(input || "").replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

async function waitUntil(fn, timeoutMs, intervalMs = 240) {
  const started = Date.now();
  while (Date.now() - started < timeoutMs) {
    const result = await fn();
    if (result) {
      return result;
    }
    await sleep(intervalMs);
  }
  return null;
}

async function switchProject(page, projectName) {
  const btn = page.getByRole("button", { name: new RegExp(`^${escapeRegExp(projectName)}$`) }).first();
  await btn.waitFor({ state: "visible", timeout: 22000 });
  await btn.click();
  await sleep(1100);
}

async function openFileFromTree(page, keyword, fileName) {
  const input = page.locator('input[placeholder="回车搜索(至少2个字符)"]:visible').first();
  await input.waitFor({ state: "visible", timeout: 22000 });
  await input.fill("");
  await input.fill(String(keyword || fileName || ""));
  await input.press("Enter");
  await sleep(800);
  const exact = new RegExp(`^${escapeRegExp(fileName)}$`);
  const title = page.locator(".workspace-source-tree .ant-tree-title:visible").filter({ hasText: exact }).first();
  await title.waitFor({ state: "visible", timeout: 18000 });
  await title.click();
  await sleep(1300);
}

async function getVisibleEditorPath(page) {
  return page.evaluate(() => {
    const root = document.querySelector("[data-viewer-id][data-store-current-file-path]");
    return String(root?.getAttribute?.("data-store-current-file-path") || "");
  });
}

async function waitForEditorPath(page, expectedPath, timeoutMs = 22000) {
  const normalizedExpected = normalizePath(expectedPath);
  return waitUntil(async () => {
    const currentPath = normalizePath(await getVisibleEditorPath(page));
    if (currentPath && currentPath.includes(normalizedExpected)) {
      return currentPath;
    }
    return null;
  }, timeoutMs, 260);
}

async function insertProbeLineAndSuggest(page, anchorRegexText, probeLineText) {
  const prepared = await page.evaluate(({ anchorRegexText, probeLineText }) => {
    const monacoNs = window?.monaco;
    const editors = monacoNs?.editor?.getEditors?.() || [];
    const getVisible = () => {
      for (const editor of editors) {
        try {
          const host = editor?.getContainerDomNode?.();
          const slotEl = host?.closest?.("[data-slot-id]");
          if (!slotEl) continue;
          const style = window.getComputedStyle(slotEl);
          if (!style || style.display === "none" || style.visibility === "hidden") continue;
          const model = editor?.getModel?.();
          if (!model) continue;
          return { editor, model };
        } catch (_error) {
          // ignore
        }
      }
      return null;
    };

    const pair = getVisible();
    if (!pair) {
      return { ok: false, reason: "visible editor not found" };
    }
    const { editor, model } = pair;
    const maxLine = Number(model.getLineCount?.() || 0);
    const re = new RegExp(anchorRegexText);
    for (let i = 1; i <= maxLine; i += 1) {
      const text = String(model.getLineContent?.(i) || "");
      if (!re.test(text)) {
        continue;
      }
      const insertLine = i + 1;
      editor.executeEdits?.("value-completion-probe", [{
        range: {
          startLineNumber: insertLine,
          startColumn: 1,
          endLineNumber: insertLine,
          endColumn: 1,
        },
        text: `${probeLineText}\n`,
      }]);
      const col = String(probeLineText || "").length + 1;
      editor.revealLineInCenter?.(insertLine);
      editor.setPosition?.({ lineNumber: insertLine, column: col });
      editor.focus?.();
      editor.trigger("keyboard", "editor.action.triggerSuggest", {});
      return { ok: true, lineNumber: insertLine, lineText: probeLineText, column: col };
    }
    return { ok: false, reason: `anchor not found: ${anchorRegexText}` };
  }, { anchorRegexText, probeLineText });

  if (!prepared?.ok) {
    throw new Error(prepared?.reason || "insert probe line failed");
  }

  const labels = await waitUntil(async () => {
    const textList = await page.locator(".suggest-widget.visible .monaco-list-row .label-name").allTextContents().catch(() => []);
    const clean = textList.map((item) => String(item || "").trim()).filter(Boolean);
    if (clean.length > 0) {
      return clean;
    }
    return null;
  }, 12000, 120);

  await page.keyboard.press("Escape").catch(() => undefined);
  await sleep(160);

  if (!labels || labels.length <= 0) {
    throw new Error(`no suggestions for probe line: ${probeLineText}`);
  }

  return {
    lineNumber: prepared.lineNumber,
    lineText: prepared.lineText,
    labels,
  };
}

function assertInclude(labels, expected, contextName) {
  if (!Array.isArray(labels) || labels.indexOf(expected) < 0) {
    throw new Error(`${contextName} missing expected suggestion: ${expected}; got: ${JSON.stringify(labels.slice(0, 20))}`);
  }
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const summary = {
    pageUrl: PAGE_URL,
    apiUrl: API_URL,
    startedAt: new Date().toISOString(),
    pass: false,
    error: "",
    screenshot: "",
    python: {
      filePath: "",
      checks: {},
    },
    go: {
      filePath: "",
      checks: {},
    },
  };

  let browser;
  try {
    const pyQuery = runCurl("webshell.workspace_file_query", {
      project_code: PY_PROJECT_CODE,
      keyword: "jira/comments/index.yml",
      pagination: false,
    });
    const pyPath = String((pyQuery.data || []).find((row) => String(row?.path || "").endsWith("/jira/comments/index.yml"))?.path || (pyQuery.data || [])[0]?.path || "");
    if (!pyPath) {
      throw new Error("python target file not found: jira/comments/index.yml");
    }
    summary.python.filePath = pyPath;

    const goQuery = runCurl("webshell.workspace_file_query", {
      project_code: GO_PROJECT_CODE,
      keyword: "config/sync/index.yml",
      pagination: false,
    });
    const goPath = String((goQuery.data || []).find((row) => String(row?.path || "").endsWith("/collect/config/sync/index.yml"))?.path || (goQuery.data || [])[0]?.path || "");
    if (!goPath) {
      throw new Error("go target file not found: collect/config/sync/index.yml");
    }
    summary.go.filePath = goPath;

    browser = await chromium.launch({ headless: true });

    const pyPage = await browser.newPage({ viewport: { width: 1680, height: 980 } });
    await pyPage.goto(PAGE_URL, { waitUntil: "domcontentloaded", timeout: 60000 });
    await pyPage.waitForTimeout(1800);
    await switchProject(pyPage, PY_PROJECT_NAME);
    await openFileFromTree(pyPage, "jira/comments/index.yml", "index.yml");
    if (!(await waitForEditorPath(pyPage, pyPath, 22000))) {
      throw new Error("python file not opened");
    }

    const pySql = await insertProbeLineAndSuggest(pyPage, "^\\s*count_sql:\\s*count\\.sql\\s*$", "    sql_file: ");
    assertInclude(pySql.labels, "issue_commit_detail.sql", "python sql_file");

    const pyDataJson = await insertProbeLineAndSuggest(pyPage, "^\\s*data_json:\\s*issue_modify\\.json\\s*$", "        data_json: ");
    assertInclude(pyDataJson.labels, "issue_modify.json", "python data_json");

    const pyDataFile = await insertProbeLineAndSuggest(pyPage, "^\\s*data_json:\\s*issue_modify\\.json\\s*$", "        data_file: ");
    assertInclude(pyDataFile.labels, "count.sql", "python data_file");

    const pyModel = await insertProbeLineAndSuggest(pyPage, "^\\s*model:\\s*IssueCommit\\s*$", "    model: Issue");
    assertInclude(pyModel.labels, "IssueCommit", "python model");

    const pyTable = await insertProbeLineAndSuggest(pyPage, "^\\s*model:\\s*IssueCommit\\s*$", "    table: alert_");
    assertInclude(pyTable.labels, "alert_ignore", "python table");

    summary.python.checks = {
      sql_file: pySql,
      data_json: pyDataJson,
      data_file: pyDataFile,
      model: pyModel,
      table: pyTable,
    };

    const goPage = await browser.newPage({ viewport: { width: 1680, height: 980 } });
    await goPage.goto(PAGE_URL, { waitUntil: "domcontentloaded", timeout: 60000 });
    await goPage.waitForTimeout(1800);
    await switchProject(goPage, GO_PROJECT_NAME);
    await openFileFromTree(goPage, "config/sync/index.yml", "index.yml");
    if (!(await waitForEditorPath(goPage, goPath, 22000))) {
      throw new Error("go file not opened");
    }

    const goModify = await insertProbeLineAndSuggest(goPage, "^\\s*modify_config:\\s*doc_modify\\.json\\s*$", "    modify_config: ");
    assertInclude(goModify.labels, "doc_modify.json", "go modify_config");

    const goDataFile = await insertProbeLineAndSuggest(goPage, "^\\s*modify_config:\\s*doc_modify\\.json\\s*$", "    data_file: ");
    assertInclude(goDataFile.labels, "doc_modify.json", "go data_file");

    const goTable = await insertProbeLineAndSuggest(goPage, "^\\s*table:\\s*collect_doc_important\\s*$", "    table: collect_doc_i");
    assertInclude(goTable.labels, "collect_doc_important", "go table");

    const goModel = await insertProbeLineAndSuggest(goPage, "^\\s*table:\\s*collect_doc_important\\s*$", "    model: CollectDocI");
    assertInclude(goModel.labels, "CollectDocImportant", "go model");

    summary.go.checks = {
      modify_config: goModify,
      data_file: goDataFile,
      table: goTable,
      model: goModel,
    };

    const shot = path.join(OUT_DIR, "webshell-editor-pool-value-completion-check.png");
    await goPage.screenshot({ path: shot, fullPage: true });
    summary.screenshot = shot;

    await goPage.close().catch(() => {});
    await pyPage.close().catch(() => {});
    summary.pass = true;
  } catch (error) {
    summary.error = String(error?.message || error);
  } finally {
    if (browser) {
      await browser.close().catch(() => {});
    }
    summary.endedAt = new Date().toISOString();
    const outFile = path.join(OUT_DIR, "webshell-editor-pool-value-completion-check.json");
    fs.writeFileSync(outFile, JSON.stringify(summary, null, 2));
    console.log(JSON.stringify({
      pass: summary.pass,
      outFile,
      screenshot: summary.screenshot,
      error: summary.error,
    }, null, 2));
    process.exit(summary.pass ? 0 : 1);
  }
})();
