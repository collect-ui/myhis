#!/usr/bin/env node

const fs = require("fs");
const path = require("path");
const { spawnSync } = require("child_process");
const { chromium } = require("playwright");

const PAGE_URL = process.env.WEBSHELL_EDITOR_POOL_PAGE_URL || "http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool";
const API_URL = process.env.WEBSHELL_EDITOR_POOL_API_URL || "http://127.0.0.1:8015/template_data/data";
const PROJECT_CODE = process.env.WEBSHELL_EDITOR_POOL_PROJECT_CODE || "backend";
const PROJECT_NAME = process.env.WEBSHELL_EDITOR_POOL_PROJECT_NAME || "月神后端";
const OUT_DIR = process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR || "/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation";

const CHECK_KEYS = [
  { key: "arr2obj", expectedSuffix: "/collect/service_imp/result_handlers/handlers/array_2_obj.py" },
  { key: "param2result", expectedSuffix: "/collect/service_imp/result_handlers/handlers/param2result.py" },
  { key: "add_param", expectedSuffix: "/collect/service_imp/result_handlers/handlers/add_param.py" },
];

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

function probeFileReadable(projectCode, filePath) {
  const payload = JSON.stringify({
    service: "webshell.workspace_file_content",
    project_code: projectCode,
    path: String(filePath || ""),
    max_bytes: 128,
  });
  const res = spawnSync("curl", [
    "--noproxy",
    "*",
    "-sS",
    "-m",
    "40",
    `${API_URL}?service=webshell.workspace_file_content`,
    "-H",
    "Content-Type: application/json",
    "--data",
    payload,
  ], { encoding: "utf8" });
  if (res.status !== 0) {
    return false;
  }
  try {
    const obj = JSON.parse(String(res.stdout || "{}"));
    return String(obj?.code || "") === "0" && obj?.success !== false;
  } catch (_error) {
    return false;
  }
}

function normalizePath(input) {
  return String(input || "").replace(/\\/g, "/").replace(/\/+$/g, "");
}

function joinPath(base, rel) {
  const left = normalizePath(base || "");
  const right = normalizePath(rel || "");
  if (!left) {
    return right;
  }
  if (!right) {
    return left;
  }
  if (right.startsWith("/")) {
    return right;
  }
  return normalizePath(`${left}/${right}`);
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

async function getVisibleEditorState(page) {
  return page.evaluate(() => {
    const rootEl = document.querySelector("[data-viewer-id][data-store-current-file-path]");
    const currentFilePath = String(rootEl?.getAttribute?.("data-store-current-file-path") || "");
    const monacoNs = window?.monaco;
    const editors = monacoNs?.editor?.getEditors?.() || [];
    for (const editor of editors) {
      try {
        const host = editor?.getContainerDomNode?.();
        const slotEl = host?.closest?.("[data-slot-id]");
        if (!slotEl) {
          continue;
        }
        const style = window.getComputedStyle(slotEl);
        if (!style || style.display === "none" || style.visibility === "hidden") {
          continue;
        }
        const model = editor?.getModel?.();
        if (!model) {
          continue;
        }
        const position = editor.getPosition?.() || { lineNumber: 1, column: 1 };
        return {
          currentFilePath,
          lineNumber: Number(position.lineNumber || 1),
          column: Number(position.column || 1),
          lineCount: Number(model.getLineCount?.() || 0),
        };
      } catch (_error) {
        // ignore
      }
    }
    return null;
  });
}

async function waitForEditorPath(page, expectedPath, timeoutMs = 22000) {
  const normalizedExpected = normalizePath(expectedPath);
  return waitUntil(async () => {
    const state = await getVisibleEditorState(page);
    if (!state) {
      return null;
    }
    const normalizedCurrent = normalizePath(state.currentFilePath || "");
    if (normalizedCurrent && normalizedCurrent.includes(normalizedExpected)) {
      return state;
    }
    return null;
  }, timeoutMs, 260);
}

async function openFileFromTree(page, keyword, fileName) {
  const input = page.locator('input[placeholder="回车搜索(至少2个字符)"]:visible').first();
  await input.waitFor({ state: "visible", timeout: 20000 });
  await input.fill("");
  await input.fill(String(keyword || ""));
  await input.press("Enter");
  await sleep(800);
  const exact = new RegExp(`^${escapeRegExp(fileName)}$`);
  const title = page.locator(".workspace-source-tree .ant-tree-title").filter({ hasText: exact }).first();
  await title.waitFor({ state: "visible", timeout: 18000 });
  await title.click();
  await sleep(1400);
}

async function clickResultHandlerKeyInEditor(page, key) {
  const result = await page.evaluate(({ key }) => {
    const getVisibleEditor = () => {
      const monacoNs = window?.monaco;
      const editors = monacoNs?.editor?.getEditors?.() || [];
      for (const editor of editors) {
        try {
          const host = editor?.getContainerDomNode?.();
          const slotEl = host?.closest?.("[data-slot-id]");
          if (!slotEl) {
            continue;
          }
          const style = window.getComputedStyle(slotEl);
          if (!style || style.display === "none" || style.visibility === "hidden") {
            continue;
          }
          const model = editor?.getModel?.();
          if (!model) {
            continue;
          }
          return { editor, model };
        } catch (_error) {
          // ignore
        }
      }
      return null;
    };

    const pair = getVisibleEditor();
    if (!pair) {
      return { ok: false, reason: "visible editor not found" };
    }
    const { editor, model } = pair;
    const maxLine = Number(model.getLineCount?.() || 0);
    const keyRe = new RegExp(`^\\s*-\\s*key\\s*:\\s*(?:'${key.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}'|\"${key.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}\"|${key.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")})\\s*(?:#.*)?$`);
    const sectionRe = /^\s*result_handler\s*:/;

    const countIndent = (line) => {
      const m = String(line || "").match(/^(\s*)/);
      return m ? m[1].length : 0;
    };

    let hitLine = 0;
    let hitColumn = 0;
    let hitText = "";
    for (let i = 1; i <= maxLine; i += 1) {
      const text = String(model.getLineContent?.(i) || "");
      if (!keyRe.test(text)) {
        continue;
      }
      const keyIndent = countIndent(text);
      let foundResultSection = false;
      for (let j = i - 1; j >= Math.max(1, i - 220); j -= 1) {
        const prevText = String(model.getLineContent?.(j) || "");
        const prevTrim = prevText.trim();
        if (!prevTrim || prevTrim.startsWith("#")) {
          continue;
        }
        const prevIndent = countIndent(prevText);
        if (sectionRe.test(prevText) && prevIndent < keyIndent) {
          foundResultSection = true;
          break;
        }
      }
      if (!foundResultSection) {
        continue;
      }
      const valueIndex = text.indexOf(key);
      if (valueIndex < 0) {
        continue;
      }
      hitLine = i;
      hitColumn = valueIndex + 2;
      hitText = text;
      break;
    }

    if (!hitLine || !hitColumn) {
      return { ok: false, reason: `result_handler key not found: ${key}` };
    }

    editor.revealLineInCenter?.(hitLine);
    editor.setPosition?.({ lineNumber: hitLine, column: hitColumn });
    editor.focus?.();

    const visible = editor.getScrolledVisiblePosition?.({ lineNumber: hitLine, column: hitColumn });
    const dom = editor.getContainerDomNode?.();
    if (!visible || !dom) {
      return { ok: false, reason: "visible position unavailable" };
    }
    const rect = dom.getBoundingClientRect();
    return {
      ok: true,
      lineNumber: hitLine,
      column: hitColumn,
      lineText: hitText,
      x: rect.left + visible.left + 4,
      y: rect.top + visible.top + Math.max(6, Math.floor(visible.height / 2)),
    };
  }, { key });

  if (!result || !result.ok) {
    throw new Error(result?.reason || `click target not found: ${key}`);
  }

  let ctrlHoverHint = false;
  await page.keyboard.down("Control");
  try {
    await page.mouse.move(Number(result.x), Number(result.y));
    await sleep(120);
    ctrlHoverHint = await page.evaluate(() => !!document.querySelector(".workspace-config-jump-link"));
    await page.mouse.click(Number(result.x), Number(result.y));
  } finally {
    await page.keyboard.up("Control");
  }
  await sleep(1400);
  return Object.assign({}, result, { ctrlHoverHint });
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const summary = {
    pageUrl: PAGE_URL,
    apiUrl: API_URL,
    projectCode: PROJECT_CODE,
    projectName: PROJECT_NAME,
    startedAt: new Date().toISOString(),
    expected: {
      indexPath: "",
      keyPathMap: {},
    },
    steps: {
      openProject: false,
      openIndex: false,
      resultHandlerArr2Obj: false,
      resultHandlerParam2Result: false,
      resultHandlerAddParam: false,
    },
    details: {},
    pass: false,
    error: "",
    screenshot: "",
  };

  let browser;
  try {
    const indexQuery = runCurl("webshell.workspace_file_query", {
      project_code: PROJECT_CODE,
      keyword: "perform/his_issue_record/index.yml",
      pagination: false,
    });
    summary.expected.indexPath = String((indexQuery.data || [])[0]?.path || "");
    if (!summary.expected.indexPath) {
      throw new Error("target index.yml not found");
    }

    const projectMetaRes = runCurl("webshell.workspace_project_query", {
      project_code: PROJECT_CODE,
      pagination: false,
    });
    const projectMeta = (projectMetaRes.data || [])[0] || {};
    const pythonPkgPath = String(projectMeta.python_pkg_path || "").trim();
    const pyRootCandidates = [];
    if (pythonPkgPath) {
      pyRootCandidates.push(normalizePath(pythonPkgPath));
      if (!/(?:site-packages|dist-packages)(?:\/|$)/.test(normalizePath(pythonPkgPath))) {
        pyRootCandidates.push(joinPath(pythonPkgPath, "lib/python2.7/site-packages"));
        pyRootCandidates.push(joinPath(pythonPkgPath, "lib/python3/site-packages"));
      }
    }
    for (const item of CHECK_KEYS) {
      const relative = item.expectedSuffix.replace(/^\/+/, "");
      let resolved = "";
      for (const root of pyRootCandidates) {
        const target = joinPath(root, relative);
        if (probeFileReadable(PROJECT_CODE, target)) {
          resolved = target;
          break;
        }
      }
      if (!resolved) {
        throw new Error(`expected implementation not found for ${item.key}`);
      }
      summary.expected.keyPathMap[item.key] = resolved;
    }

    browser = await chromium.launch({ headless: true });
    const page = await browser.newPage({ viewport: { width: 1600, height: 960 } });
    page.on("dialog", async (dialog) => {
      await dialog.dismiss().catch(() => {});
    });

    await page.goto(PAGE_URL, { waitUntil: "domcontentloaded", timeout: 60000 });
    await page.waitForTimeout(1600);

    const projectBtn = page.locator(".workspace-project-btn").filter({ hasText: new RegExp(`^${escapeRegExp(PROJECT_NAME)}$`) }).first();
    if (await projectBtn.count()) {
      await projectBtn.click();
      await page.waitForTimeout(1200);
    }
    summary.steps.openProject = true;

    await openFileFromTree(page, "his_issue_record/index.yml", "index.yml");
    const openedIndex = await waitForEditorPath(page, summary.expected.indexPath, 22000);
    if (!openedIndex) {
      throw new Error(`index file not opened: ${summary.expected.indexPath}`);
    }
    summary.steps.openIndex = true;

    for (const item of CHECK_KEYS) {
      const clickInfo = await clickResultHandlerKeyInEditor(page, item.key);
      const expectedPath = String(summary.expected.keyPathMap[item.key] || "");
      const opened = await waitForEditorPath(page, expectedPath, 22000);
      if (!opened) {
        throw new Error(`jump target mismatch for ${item.key}, expected: ${expectedPath}`);
      }
      summary.details[item.key] = {
        clickLine: clickInfo.lineNumber,
        clickColumn: clickInfo.column,
        clickText: clickInfo.lineText,
        ctrlHoverHint: clickInfo.ctrlHoverHint,
        openedPath: expectedPath,
      };
      if (item.key === "arr2obj") {
        summary.steps.resultHandlerArr2Obj = true;
      } else if (item.key === "param2result") {
        summary.steps.resultHandlerParam2Result = true;
      } else if (item.key === "add_param") {
        summary.steps.resultHandlerAddParam = true;
      }

      await openFileFromTree(page, "his_issue_record/index.yml", "index.yml");
      const back = await waitForEditorPath(page, summary.expected.indexPath, 16000);
      if (!back) {
        throw new Error("failed to return to index.yml");
      }
    }

    summary.screenshot = path.join(OUT_DIR, "webshell-editor-pool-python-result-handler-jump-check.png");
    await page.screenshot({ path: summary.screenshot, fullPage: true });
    summary.pass = true;
  } catch (error) {
    summary.error = String(error?.message || error);
  } finally {
    if (browser) {
      await browser.close().catch(() => {});
    }
    summary.endedAt = new Date().toISOString();
    const outFile = path.join(OUT_DIR, "webshell-editor-pool-python-result-handler-jump-check.json");
    fs.writeFileSync(outFile, JSON.stringify(summary, null, 2));
    console.log(JSON.stringify({
      pass: summary.pass,
      outFile,
      screenshot: summary.screenshot,
      steps: summary.steps,
      error: summary.error,
    }, null, 2));
    process.exit(summary.pass ? 0 : 1);
  }
})();
