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

async function waitUntil(fn, timeoutMs, intervalMs = 250) {
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

async function openFileFromTree(page, keyword, fileName) {
  const input = page.locator('input[placeholder="回车搜索(至少2个字符)"]:visible').first();
  await input.waitFor({ state: "visible", timeout: 22000 });
  await input.fill("");
  await input.fill(String(keyword || fileName || ""));
  await input.press("Enter");
  await sleep(760);
  const exact = new RegExp(`^${escapeRegExp(fileName)}$`);
  const title = page.locator(".workspace-source-tree .ant-tree-title:visible").filter({ hasText: exact }).first();
  await title.waitFor({ state: "visible", timeout: 18000 });
  await title.click();
  await sleep(1300);
}

async function switchProject(page, projectName) {
  const btn = page.getByRole("button", { name: new RegExp(`^${escapeRegExp(projectName)}$`) }).first();
  await btn.waitFor({ state: "visible", timeout: 20000 });
  await btn.click();
  await sleep(1200);
}

async function triggerKeySuggestAndRead(page, keyRegexText, insertKeyPrefix = "") {
  const prepared = await page.evaluate(({ keyRegexText, insertKeyPrefix }) => {
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
    const re = new RegExp(keyRegexText);
    for (let i = 1; i <= maxLine; i += 1) {
      const text = String(model.getLineContent?.(i) || "");
      if (!re.test(text)) {
        continue;
      }
      const indentMatch = text.match(/^(\s*)/);
      const indent = indentMatch ? indentMatch[1] : "";
      const insertLineText = `${indent}- key: ${insertKeyPrefix || ""}`;
      editor.executeEdits?.("lowcode-suggest", [{
        range: {
          startLineNumber: i + 1,
          startColumn: 1,
          endLineNumber: i + 1,
          endColumn: 1,
        },
        text: `${insertLineText}\n`,
      }]);
      editor.revealLineInCenter?.(i + 1);
      editor.setPosition?.({ lineNumber: i + 1, column: insertLineText.length + 1 });
      editor.focus?.();
      editor.trigger("keyboard", "editor.action.triggerSuggest", {});
      return { ok: true, lineNumber: i + 1, column: insertLineText.length + 1, lineText: insertLineText };
    }
    return { ok: false, reason: "target key line not found" };
  }, { keyRegexText, insertKeyPrefix });
  if (!prepared?.ok) {
    throw new Error(prepared?.reason || "trigger suggest failed");
  }
  await sleep(700);
  const labels = await page.locator(".suggest-widget.visible .monaco-list-row .label-name").allTextContents();
  return {
    clickLine: prepared.lineNumber,
    clickColumn: prepared.column,
    clickText: prepared.lineText,
    suggestions: labels.map((item) => String(item || "").trim()).filter(Boolean),
  };
}

async function prepareCursorAtLineEnd(page, keyRegexText) {
  const prepared = await page.evaluate(({ keyRegexText }) => {
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
    const re = new RegExp(keyRegexText);
    for (let i = 1; i <= maxLine; i += 1) {
      const text = String(model.getLineContent?.(i) || "");
      if (!re.test(text)) {
        continue;
      }
      const indent = String((text.match(/^(\s*)/) || [null, ""])[1] || "");
      const maxColumn = Number(model.getLineMaxColumn?.(i) || (text.length + 1));
      editor.revealLineInCenter?.(i);
      editor.setPosition?.({ lineNumber: i, column: maxColumn });
      editor.focus?.();
      return { ok: true, lineNumber: i, indent, maxColumn };
    }
    return { ok: false, reason: "target line not found" };
  }, { keyRegexText });
  if (!prepared?.ok) {
    throw new Error(prepared?.reason || "prepare cursor failed");
  }
  return prepared;
}

async function setVisibleEditorPosition(page, lineNumber, column) {
  await page.evaluate(({ lineNumber, column }) => {
    const monacoNs = window?.monaco;
    const editors = monacoNs?.editor?.getEditors?.() || [];
    for (const editor of editors) {
      try {
        const host = editor?.getContainerDomNode?.();
        const slotEl = host?.closest?.("[data-slot-id]");
        if (!slotEl) continue;
        const style = window.getComputedStyle(slotEl);
        if (!style || style.display === "none" || style.visibility === "hidden") continue;
        const model = editor?.getModel?.();
        if (!model) continue;
        const maxLine = Number(model.getLineCount?.() || 1);
        const nextLine = Math.max(1, Math.min(maxLine, Number(lineNumber || 1)));
        const maxColumn = Number(model.getLineMaxColumn?.(nextLine) || 1);
        const nextColumn = Math.max(1, Math.min(maxColumn, Number(column || 1)));
        editor.revealLineInCenter?.(nextLine);
        editor.setPosition?.({ lineNumber: nextLine, column: nextColumn });
        editor.focus?.();
        return;
      } catch (_error) {
        // ignore
      }
    }
  }, { lineNumber, column });
}

async function getVisibleEditorLineText(page, lineNumber) {
  return page.evaluate(({ lineNumber }) => {
    const monacoNs = window?.monaco;
    const editors = monacoNs?.editor?.getEditors?.() || [];
    for (const editor of editors) {
      try {
        const host = editor?.getContainerDomNode?.();
        const slotEl = host?.closest?.("[data-slot-id]");
        if (!slotEl) continue;
        const style = window.getComputedStyle(slotEl);
        if (!style || style.display === "none" || style.visibility === "hidden") continue;
        const model = editor?.getModel?.();
        if (!model) continue;
        const maxLine = Number(model.getLineCount?.() || 1);
        const nextLine = Math.max(1, Math.min(maxLine, Number(lineNumber || 1)));
        return String(model.getLineContent?.(nextLine) || "");
      } catch (_error) {
        // ignore
      }
    }
    return "";
  }, { lineNumber });
}

async function triggerSuggestAtCursor(page) {
  await page.evaluate(() => {
    const monacoNs = window?.monaco;
    const editors = monacoNs?.editor?.getEditors?.() || [];
    for (const editor of editors) {
      try {
        const host = editor?.getContainerDomNode?.();
        const slotEl = host?.closest?.("[data-slot-id]");
        if (!slotEl) continue;
        const style = window.getComputedStyle(slotEl);
        if (!style || style.display === "none" || style.visibility === "hidden") continue;
        const model = editor?.getModel?.();
        if (!model) continue;
        editor.focus?.();
        editor.trigger("keyboard", "editor.action.triggerSuggest", {});
        return;
      } catch (_error) {
        // ignore
      }
    }
  });
}

async function readVisibleSuggestLabels(page) {
  const labels = await page.locator(".suggest-widget.visible .monaco-list-row .label-name").allTextContents();
  return labels.map((item) => String(item || "").trim()).filter(Boolean);
}

async function insertListPrefixLineAndFocus(page, anchorLineNumber, listPrefix) {
  return page.evaluate(({ anchorLineNumber, listPrefix }) => {
    const monacoNs = window?.monaco;
    const editors = monacoNs?.editor?.getEditors?.() || [];
    for (const editor of editors) {
      try {
        const host = editor?.getContainerDomNode?.();
        const slotEl = host?.closest?.("[data-slot-id]");
        if (!slotEl) continue;
        const style = window.getComputedStyle(slotEl);
        if (!style || style.display === "none" || style.visibility === "hidden") continue;
        const model = editor?.getModel?.();
        if (!model) continue;
        const maxLine = Number(model.getLineCount?.() || 1);
        const targetLine = Math.max(1, Math.min(maxLine, Number(anchorLineNumber || 1)));
        const insertLine = targetLine + 1;
        editor.executeEdits?.("lowcode-insert-list-prefix", [{
          range: {
            startLineNumber: insertLine,
            startColumn: 1,
            endLineNumber: insertLine,
            endColumn: 1,
          },
          text: `${listPrefix}\n`,
        }]);
        editor.revealLineInCenter?.(insertLine);
        editor.setPosition?.({ lineNumber: insertLine, column: String(listPrefix || "").length + 1 });
        editor.focus?.();
        return {
          ok: true,
          lineNumber: insertLine,
          lineText: String(model.getLineContent?.(insertLine) || ""),
        };
      } catch (_error) {
        // ignore
      }
    }
    return { ok: false, reason: "visible editor not found for insert list line" };
  }, { anchorLineNumber, listPrefix });
}

async function verifyYamlArrayEnterAndFirstKeySuggest(page, keyRegexText) {
  const prepared = await prepareCursorAtLineEnd(page, keyRegexText);
  await page.keyboard.press("Enter");
  await sleep(520);

  const nextLine = prepared.lineNumber + 1;
  const nextLineText = await getVisibleEditorLineText(page, nextLine);
  const expectedChildIndent = `${prepared.indent}  `;
  if (!nextLineText.startsWith(expectedChildIndent)) {
    throw new Error(`yaml enter should keep object-indent; expected prefix "${expectedChildIndent}" got "${nextLineText}"`);
  }

  const expectedListPrefix = `${prepared.indent}- `;
  let keySuggestLine = nextLine;
  let keySuggestLineText = nextLineText;
  if (nextLineText !== expectedListPrefix) {
    const inserted = await insertListPrefixLineAndFocus(page, nextLine, expectedListPrefix);
    if (!inserted?.ok) {
      throw new Error(inserted?.reason || "insert list prefix line failed");
    }
    keySuggestLine = Number(inserted.lineNumber || (nextLine + 1));
    keySuggestLineText = String(inserted.lineText || "");
  } else {
    await setVisibleEditorPosition(page, keySuggestLine, expectedListPrefix.length + 1);
  }

  await triggerSuggestAtCursor(page);
  await sleep(680);
  const firstKeySuggestions = await readVisibleSuggestLabels(page);
  if (!firstKeySuggestions.includes("key")) {
    throw new Error("first key suggestions missing: key");
  }
  if (firstKeySuggestions.length < 4) {
    throw new Error(`first key suggestions too few: ${firstKeySuggestions.length}`);
  }
  await page.keyboard.press("Escape");
  await sleep(180);

  await setVisibleEditorPosition(page, keySuggestLine, expectedListPrefix.length + 1);
  await page.keyboard.type("k", { delay: 20 });
  const autoKeySuggestions = await waitUntil(async () => {
    const labels = await readVisibleSuggestLabels(page);
    if (labels.length > 0) {
      return labels;
    }
    return null;
  }, 2200, 120);
  if (!autoKeySuggestions || !autoKeySuggestions.includes("key")) {
    throw new Error("typing k should auto show key suggestion");
  }
  await page.keyboard.press("Escape");
  await sleep(180);

  return {
    targetLine: prepared.lineNumber,
    insertedLine: nextLine,
    insertedLineText: nextLineText,
    keySuggestLine,
    keySuggestLineText,
    expectedListPrefix,
    firstKeySuggestions,
    autoKeySuggestions,
  };
}

async function readShortcutHintInfo(page) {
  const el = page.locator("span").filter({ hasText: /^快捷键提示$/ }).first();
  await el.waitFor({ state: "visible", timeout: 12000 });
  const lowcodeHintCount = await page.locator("span").filter({ hasText: /^低代码提示$/ }).count().catch(() => 0);
  return {
    title: String((await el.getAttribute("title")) || ""),
    hasStandaloneLowcodeHint: Number(lowcodeHintCount || 0) > 0,
  };
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const summary = {
    pageUrl: PAGE_URL,
    apiUrl: API_URL,
    startedAt: new Date().toISOString(),
    expected: {
      pythonKeys: ["arr2obj", "add_param"],
      goKeys: ["service2field"],
    },
    steps: {
      pythonHint: false,
      pythonSuggest: false,
      pythonArrayEnter: false,
      pythonFirstKeySuggest: false,
      pythonAutoKeySuggest: false,
      goHint: false,
      goSuggest: false,
      goIsolation: false,
    },
    details: {
      python: {},
      go: {},
    },
    pass: false,
    error: "",
    screenshot: "",
  };

  let browser;
  try {
    const pyFileRes = runCurl("webshell.workspace_file_query", {
      project_code: PY_PROJECT_CODE,
      keyword: "perform/his_issue_record/index.yml",
      pagination: false,
    });
    const pyPath = String((pyFileRes.data || [])[0]?.path || "");
    if (!pyPath) {
      throw new Error("python target file not found");
    }
    const goFileRes = runCurl("webshell.workspace_file_query", {
      project_code: GO_PROJECT_CODE,
      keyword: "service_router.yml",
      pagination: false,
    });
    const goPath = String((goFileRes.data || [])[0]?.path || "");
    if (!goPath) {
      throw new Error("go target file not found");
    }

    browser = await chromium.launch({ headless: true });
    const page = await browser.newPage({ viewport: { width: 1600, height: 960 } });
    await page.goto(PAGE_URL, { waitUntil: "domcontentloaded", timeout: 60000 });
    await page.waitForTimeout(1700);

    await switchProject(page, PY_PROJECT_NAME);
    await openFileFromTree(page, "his_issue_record/index.yml", "index.yml");
    if (!(await waitForEditorPath(page, pyPath, 22000))) {
      throw new Error("python file not opened");
    }
    const pyHintInfo = await readShortcutHintInfo(page);
    summary.details.python.hintTitle = pyHintInfo.title;
    summary.details.python.hasStandaloneLowcodeHint = pyHintInfo.hasStandaloneLowcodeHint;
    if (pyHintInfo.hasStandaloneLowcodeHint) {
      throw new Error("python should not show standalone 低代码提示 tag");
    }
    if (!pyHintInfo.title.includes("Python低代码提示")) {
      throw new Error("python hint text mismatch");
    }
    summary.steps.pythonHint = true;

    const pySuggest = await triggerKeySuggestAndRead(page, "^\\s*-\\s*key\\s*:\\s*add_param\\s*$");
    summary.details.python.suggest = pySuggest;
    for (const key of summary.expected.pythonKeys) {
      if (!pySuggest.suggestions.includes(key)) {
        throw new Error(`python suggest missing key: ${key}`);
      }
    }
    summary.steps.pythonSuggest = true;
    await page.keyboard.press("Escape");
    await sleep(220);

    const pyArrayAndKey = await verifyYamlArrayEnterAndFirstKeySuggest(page, "^\\s*-\\s*key\\s*:\\s*add_param\\s*$");
    summary.details.python.arrayEnter = pyArrayAndKey;
    summary.steps.pythonArrayEnter = true;
    summary.steps.pythonFirstKeySuggest = true;
    summary.steps.pythonAutoKeySuggest = true;

    const goPage = await browser.newPage({ viewport: { width: 1600, height: 960 } });
    await goPage.goto(PAGE_URL, { waitUntil: "domcontentloaded", timeout: 60000 });
    await goPage.waitForTimeout(1700);
    await switchProject(goPage, GO_PROJECT_NAME);
    await openFileFromTree(goPage, "service_router.yml", "service_router.yml");
    if (!(await waitForEditorPath(goPage, goPath, 22000))) {
      throw new Error("go file not opened");
    }
    const goHintInfo = await readShortcutHintInfo(goPage);
    summary.details.go.hintTitle = goHintInfo.title;
    summary.details.go.hasStandaloneLowcodeHint = goHintInfo.hasStandaloneLowcodeHint;
    if (goHintInfo.hasStandaloneLowcodeHint) {
      throw new Error("go should not show standalone 低代码提示 tag");
    }
    if (!goHintInfo.title.includes("Go低代码提示")) {
      throw new Error("go hint text mismatch");
    }
    summary.steps.goHint = true;

    const goSuggest = await triggerKeySuggestAndRead(goPage, "^\\s*-\\s*key\\s*:\\s*service2field\\s*$", "serv");
    summary.details.go.suggest = goSuggest;
    for (const key of summary.expected.goKeys) {
      if (!goSuggest.suggestions.includes(key)) {
        throw new Error(`go suggest missing key: ${key}`);
      }
    }
    summary.steps.goSuggest = true;
    if (goSuggest.suggestions.includes("add_param")) {
      throw new Error("go suggest should not include python-only key add_param");
    }
    summary.steps.goIsolation = true;

    summary.screenshot = path.join(OUT_DIR, "webshell-editor-pool-lowcode-completion-check.png");
    await goPage.screenshot({ path: summary.screenshot, fullPage: true });
    await goPage.close().catch(() => {});
    summary.pass = true;
  } catch (error) {
    summary.error = String(error?.message || error);
  } finally {
    if (browser) {
      await browser.close().catch(() => {});
    }
    summary.endedAt = new Date().toISOString();
    const outFile = path.join(OUT_DIR, "webshell-editor-pool-lowcode-completion-check.json");
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
