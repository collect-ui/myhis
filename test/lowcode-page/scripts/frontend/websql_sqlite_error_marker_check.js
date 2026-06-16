#!/usr/bin/env node

const fs = require("fs");
const path = require("path");

let playwright;
try {
  playwright = require("playwright");
} catch (_error) {
  playwright = require("/data/project/sport-ui/node_modules/playwright");
}

const { chromium } = playwright;

const BASE_URL = process.env.WEBSQL_BASE_URL || "http://192.168.232.130:8015";
const TARGET_URL =
  process.env.WEBSQL_TARGET_URL ||
  `${BASE_URL}/collect-ui#/collect-ui/framework/websql-pool`;
const OUT_DIR =
  process.env.WEBSQL_OUTPUT_DIR ||
  "/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation";

const CASES = [
  {
    name: "sqlite-syntax",
    sql: "SELECT name, type FROM  WHERE type IN ('table','view') ORDER BY type, name;",
    expectedToken: "WHERE",
    expectedKind: "syntax_error",
    expectedLine: 1,
    expectedColumn: 25,
    expectedDirectionLabel: "重点检查：标记前方",
    expectedRawText: 'near "WHERE": syntax error',
    expectedPanelText: ["错误信息：", "问题侧：重点检查：标记前方", "标记前：SELECT name, type FROM", "标记后：type IN"],
  },
  {
    name: "sqlite-field",
    sql: "SELECT definitely_missing_col FROM sqlite_master;",
    expectedToken: "definitely_missing_col",
    expectedKind: "unknown_column",
    expectedLine: 1,
    expectedColumn: 8,
    expectedDirectionLabel: "问题点：标记字段",
    expectedRawText: "no such column: definitely_missing_col",
    expectedPanelText: ["错误信息：", "问题侧：问题点：标记字段", "提示：字段", "标记后：FROM sqlite_master"],
  },
  {
    name: "sqlite-multiline-where-field",
    sql: "select *\nfrom attachment\nwhere attachment_id1=1",
    expectedToken: "attachment_id1",
    expectedKind: "unknown_column",
    expectedLine: 3,
    expectedColumn: 7,
    expectedDirectionLabel: "问题点：标记字段",
    expectedRawText: "no such column: attachment_id1",
    expectedPanelText: ["错误信息：", "位置：第 3 行，第 7 列", "问题侧：问题点：标记字段", "标记前：select * from attachment where"],
  },
  {
    name: "sqlite-button-leading-blank-field",
    editorSql: "\n\nselect * from attachment where attachment_id1=1\n\n",
    sql: "select * from attachment where attachment_id1=1",
    expectedSelectionStartLine: 3,
    expectedSelectionStartColumn: 1,
    expectedToken: "attachment_id1",
    expectedKind: "unknown_column",
    expectedLine: 3,
    expectedColumn: 32,
    expectedDirectionLabel: "问题点：标记字段",
    expectedRawText: "no such column: attachment_id1",
    expectedPanelText: ["错误信息：", "位置：第 3 行，第 32 列", "问题侧：问题点：标记字段", "标记前：select * from attachment where"],
  },
  {
    name: "sqlite-selected-line-field",
    editorSql: "select 1;\nselect 2;\nselect attachment_id1 from attachment;",
    sql: "select attachment_id1 from attachment;",
    selectedSql: "select attachment_id1 from attachment;",
    expectedSelectionStartLine: 3,
    expectedSelectionStartColumn: 1,
    expectedToken: "attachment_id1",
    expectedKind: "unknown_column",
    expectedLine: 3,
    expectedColumn: 8,
    expectedDirectionLabel: "问题点：标记字段",
    expectedRawText: "no such column: attachment_id1",
    expectedPanelText: ["错误信息：", "位置：第 3 行，第 8 列", "问题侧：问题点：标记字段", "标记前：select"],
  },
  {
    name: "sqlite-multiline-select-list-field",
    sql: "select\n  attachment_id,\n  filename,\n  attachment_id1,\n  path\nfrom attachment;",
    expectedToken: "attachment_id1",
    expectedKind: "unknown_column",
    expectedLine: 4,
    expectedColumn: 3,
    expectedDirectionLabel: "问题点：标记字段",
    expectedRawText: "no such column: attachment_id1",
    expectedPanelText: ["错误信息：", "位置：第 4 行，第 3 列", "问题侧：问题点：标记字段", "标记后：, path from attachment"],
  },
  {
    name: "sqlite-multiline-syntax-before-where",
    sql: "SELECT attachment_id\nFROM\nWHERE attachment_id=1;",
    expectedToken: "WHERE",
    expectedKind: "syntax_error",
    expectedLine: 3,
    expectedColumn: 1,
    expectedDirectionLabel: "重点检查：标记前方",
    expectedRawText: 'near "WHERE": syntax error',
    expectedPanelText: ["错误信息：", "位置：第 3 行，第 1 列", "问题侧：重点检查：标记前方", "标记前：SELECT attachment_id FROM"],
  },
  {
    name: "sqlite-multiline-table",
    sql: "select attachment_id\nfrom attachment_missing\nwhere attachment_id=1;",
    expectedToken: "attachment_missing",
    expectedKind: "unknown_table",
    expectedLine: 2,
    expectedColumn: 6,
    expectedDirectionLabel: "问题点：标记表名",
    expectedRawText: "no such table: attachment_missing",
    expectedPanelText: ["错误信息：", "位置：第 2 行，第 6 列", "问题侧：问题点：标记表名", "标记前：select attachment_id from"],
  },
];

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function assertCheck(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function parseRequestData(request) {
  try {
    return JSON.parse(request.postData() || "{}");
  } catch (_error) {
    return {};
  }
}

async function clearWebSQLLocalState(page) {
  await page.evaluate(() => {
    const prefixes = [
      "workspace-websql-connections",
      "workspace-websql-recent",
      "workspace-websql-favorites",
      "websql-lowcode",
      "workspace-websql-panel-state",
    ];
    for (const key of Object.keys(window.localStorage || {})) {
      if (prefixes.some((prefix) => key.startsWith(prefix))) {
        window.localStorage.removeItem(key);
      }
    }
  });
}

async function gotoPage(page, url) {
  if (page.url() === url) {
    await page.reload({ waitUntil: "domcontentloaded", timeout: 45000 });
  } else {
    await page.goto(url, { waitUntil: "domcontentloaded", timeout: 45000 });
  }
  await sleep(2500);
}

async function setWebSQLEditorValue(page, value) {
  const result = await page.evaluate((nextValue) => {
    const editors = window?.monaco?.editor?.getEditors?.() || [];
    for (const editor of editors) {
      const node = editor?.getContainerDomNode?.();
      if (!node || !node.closest(".websql-lowcode")) continue;
      const model = editor?.getModel?.();
      if (!model) continue;
      editor.setValue(String(nextValue || ""));
      const line = model.getLineCount?.() || 1;
      const column = model.getLineMaxColumn?.(line) || 1;
      if (window.monaco?.Selection) {
        editor.setSelection(new window.monaco.Selection(line, column, line, column));
      }
      editor.focus();
      return { ok: true, uri: String(model.uri || "") };
    }
    return { ok: false, reason: "websql monaco editor not found" };
  }, value);
  assertCheck(result.ok, result.reason || "failed to set WebSQL editor value");
  await sleep(800);
  return result;
}

async function selectWebSQLEditorText(page, text) {
  const result = await page.evaluate((selectedText) => {
    const editors = window?.monaco?.editor?.getEditors?.() || [];
    const editor = editors.find((item) => {
      const node = item?.getContainerDomNode?.();
      return node && node.closest(".websql-lowcode");
    });
    const model = editor?.getModel?.();
    if (!editor || !model) {
      return { ok: false, reason: "websql monaco editor not found" };
    }
    const value = String(model.getValue?.() || "");
    const start = value.indexOf(String(selectedText || ""));
    if (start < 0) {
      return { ok: false, reason: "selected text not found" };
    }
    const startPos = model.getPositionAt(start);
    const endPos = model.getPositionAt(start + String(selectedText || "").length);
    const selection = new window.monaco.Selection(
      startPos.lineNumber,
      startPos.column,
      endPos.lineNumber,
      endPos.column
    );
    editor.focus();
    editor.setSelection(selection);
    editor.revealRangeInCenter?.(selection);
    return {
      ok: true,
      selectedText: String(model.getValueInRange(selection) || ""),
      selection: {
        startLineNumber: selection.startLineNumber,
        startColumn: selection.startColumn,
        endLineNumber: selection.endLineNumber,
        endColumn: selection.endColumn,
      },
    };
  }, text);
  assertCheck(result.ok, result.reason || "failed to select WebSQL editor text");
  assertCheck(
    String(result.selectedText || "").trim() === String(text || "").trim(),
    `selected text mismatch: ${JSON.stringify(result)}`
  );
  await sleep(800);
  return result;
}

async function clickExecute(page) {
  const button = page.locator(".websql-execute-btn").first();
  await button.waitFor({ state: "visible", timeout: 15000 });
  await button.click();
}

async function waitForExecutionResponse(page, trigger, sql) {
  const responsePromise = page.waitForResponse((resp) => {
    if (!resp.url().includes("service=webshell.websql_execute")) {
      return false;
    }
    if (resp.request().method() !== "POST") {
      return false;
    }
    const payload = parseRequestData(resp.request());
    return (
      payload.operation === "execute" &&
      String(payload.sql || "").trim() === String(sql || "").trim()
    );
  }, { timeout: 30000 });
  await trigger();
  const response = await responsePromise;
  return {
    payload: parseRequestData(response.request()),
    json: await response.json(),
  };
}

async function readEditorMarkers(page) {
  return page.evaluate(() => {
    const editors = window?.monaco?.editor?.getEditors?.() || [];
    const editor = editors.find((item) => {
      const node = item?.getContainerDomNode?.();
      return node && node.closest(".websql-lowcode");
    });
    const model = editor?.getModel?.();
    if (!model) {
      return [];
    }
    return (window.monaco.editor.getModelMarkers({ resource: model.uri }) || []).map(
      (marker) => ({
        message: String(marker.message || ""),
        severity: marker.severity,
        source: String(marker.source || ""),
        startLineNumber: marker.startLineNumber,
        startColumn: marker.startColumn,
        endLineNumber: marker.endLineNumber,
        endColumn: marker.endColumn,
      })
    );
  });
}

async function readEditorErrorDecoration(page, expected) {
  return page.evaluate((expectedCase) => {
    const root = document.querySelector(".websql-lowcode");
    if (!root) {
      return { inlineCount: 0, widgetTexts: [], glyphCount: 0, widgetPosition: null };
    }
    const visibleText = (el) => {
      const rect = el.getBoundingClientRect();
      const style = window.getComputedStyle(el);
      return rect.width > 0 && rect.height > 0 && style.display !== "none";
    };
    const widgets = Array.from(
      root.querySelectorAll(".sport-ui-editor-marker-widget")
    )
      .filter(visibleText);
    const widgetTexts = widgets
      .map((el) => String(el.textContent || "").replace(/\s+/g, " ").trim())
      .filter(Boolean);
    const matchingWidget = widgets.find((el) => {
      const text = String(el.textContent || "");
      return (
        (!expectedCase?.expectedToken || text.includes(expectedCase.expectedToken)) &&
        (!expectedCase?.expectedRawText || text.includes(expectedCase.expectedRawText))
      );
    }) || widgets[0];
    const widgetRect = matchingWidget?.getBoundingClientRect?.();
    const editors = window?.monaco?.editor?.getEditors?.() || [];
    const editor = editors.find((item) => {
      const node = item?.getContainerDomNode?.();
      return node && node.closest(".websql-lowcode");
    });
    let widgetPosition = null;
    try {
      const editorNode = editor?.getDomNode?.() || editor?.getContainerDomNode?.();
      const editorRect = editorNode?.getBoundingClientRect?.();
      const lineNumber = Number(expectedCase?.expectedLine || 1);
      const column = Number(expectedCase?.expectedColumn || 1);
      const expectedPos = editor?.getScrolledVisiblePosition?.({
        lineNumber,
        column,
      });
      const firstPos = editor?.getScrolledVisiblePosition?.({
        lineNumber: 1,
        column: 1,
      });
      if (widgetRect && editorRect && expectedPos && firstPos) {
        const widgetTop = widgetRect.top;
        const expectedLineTop = editorRect.top + expectedPos.top;
        const expectedLineBottom = expectedLineTop + expectedPos.height;
        const firstLineTop = editorRect.top + firstPos.top;
        const firstLineBottom = firstLineTop + firstPos.height;
        const distanceToExpectedLine = Math.min(
          Math.abs(widgetTop - expectedLineTop),
          Math.abs(widgetTop - expectedLineBottom)
        );
        const distanceToFirstLine = Math.min(
          Math.abs(widgetTop - firstLineTop),
          Math.abs(widgetTop - firstLineBottom)
        );
        widgetPosition = {
          widgetTop,
          widgetLeft: widgetRect.left,
          expectedLineTop,
          expectedLineBottom,
          firstLineTop,
          firstLineBottom,
          distanceToExpectedLine,
          distanceToFirstLine,
        };
      }
    } catch (_error) {
      widgetPosition = null;
    }
    return {
      inlineCount: Array.from(
        root.querySelectorAll(".sport-ui-editor-marker-inline")
      ).filter(visibleText).length,
      widgetTexts,
      glyphCount: Array.from(
        root.querySelectorAll(".sport-ui-editor-marker-glyph")
      ).filter(visibleText).length,
      widgetPosition,
    };
  }, expected);
}

async function waitForMarker(page, expected) {
  const started = Date.now();
  let lastMarkers = [];
  while (Date.now() - started < 15000) {
    lastMarkers = await readEditorMarkers(page);
    const hit = lastMarkers.find((marker) => {
      return (
        Number(marker.startLineNumber) === expected.expectedLine &&
        Number(marker.startColumn) === expected.expectedColumn &&
        String(marker.message || "").includes(expected.expectedToken)
      );
    });
    if (hit) {
      return { hit, markers: lastMarkers };
    }
    await sleep(300);
  }
  throw new Error(
    `marker not found for ${expected.name}: ${JSON.stringify(lastMarkers)}`
  );
}

async function waitForEditorDecoration(page, expected) {
  const started = Date.now();
  let lastDecoration = null;
  while (Date.now() - started < 15000) {
    lastDecoration = await readEditorErrorDecoration(page, expected);
    const widgetText = (lastDecoration.widgetTexts || []).join(" ");
    if (
      lastDecoration.inlineCount > 0 &&
      widgetText.includes("↑") &&
      widgetText.includes(expected.expectedDirectionLabel) &&
      widgetText.includes(expected.expectedToken) &&
      widgetText.includes("原始 SQL 错误日志") &&
      (!expected.expectedRawText || widgetText.includes(expected.expectedRawText))
    ) {
      if (Number(expected.expectedLine) > 1) {
        const pos = lastDecoration.widgetPosition || {};
        assertCheck(
          Number.isFinite(pos.distanceToExpectedLine) &&
            Number.isFinite(pos.distanceToFirstLine),
          `${expected.name}: widget position metrics missing: ${JSON.stringify(lastDecoration)}`
        );
        assertCheck(
          pos.distanceToExpectedLine < pos.distanceToFirstLine,
          `${expected.name}: widget is closer to first line than expected line: ${JSON.stringify(pos)}`
        );
        assertCheck(
          pos.widgetTop >= pos.expectedLineTop - 6 &&
            pos.widgetTop <= pos.expectedLineBottom + 80,
          `${expected.name}: widget is not below/near expected line: ${JSON.stringify(pos)}`
        );
      }
      return lastDecoration;
    }
    await sleep(300);
  }
  throw new Error(
    `editor decoration not found for ${expected.name}: ${JSON.stringify(lastDecoration)}`
  );
}

async function readVisibleErrorPanelText(page) {
  return page.evaluate(() => {
    return Array.from(document.querySelectorAll(".websql-error"))
      .filter((el) => {
        const rect = el.getBoundingClientRect();
        const style = window.getComputedStyle(el);
        return rect.width > 0 && rect.height > 0 && style.display !== "none";
      })
      .map((el) => String(el.textContent || "").trim())
      .filter(Boolean)
      .pop() || "";
  });
}

async function closeVisibleEditorErrorWidget(page) {
  const closeButton = page
    .locator(".websql-lowcode .sport-ui-editor-marker-widget-close")
    .first();
  await closeButton.waitFor({ state: "visible", timeout: 5000 });
  const closeButtonPlacement = await page.evaluate(() => {
    const widget = document.querySelector(".websql-lowcode .sport-ui-editor-marker-widget");
    const button = document.querySelector(
      ".websql-lowcode .sport-ui-editor-marker-widget-close"
    );
    const widgetRect = widget?.getBoundingClientRect?.();
    const buttonRect = button?.getBoundingClientRect?.();
    if (!widgetRect || !buttonRect) {
      return null;
    }
    return {
      topGap: buttonRect.top - widgetRect.top,
      rightGap: widgetRect.right - buttonRect.right,
      buttonWidth: buttonRect.width,
      buttonHeight: buttonRect.height,
    };
  });
  assertCheck(
    closeButtonPlacement &&
      closeButtonPlacement.topGap >= 0 &&
      closeButtonPlacement.topGap <= 6 &&
      closeButtonPlacement.rightGap >= 0 &&
      closeButtonPlacement.rightGap <= 8,
    `close button is not top-right: ${JSON.stringify(closeButtonPlacement)}`
  );
  await closeButton.click();
  await page.waitForFunction(() => {
    const visibleText = (el) => {
      const rect = el.getBoundingClientRect();
      const style = window.getComputedStyle(el);
      return rect.width > 0 && rect.height > 0 && style.display !== "none";
    };
    return !Array.from(
      document.querySelectorAll(".websql-lowcode .sport-ui-editor-marker-widget")
    ).some(visibleText);
  }, { timeout: 5000 });
  const decorationAfterClose = await readEditorErrorDecoration(page, {});
  assertCheck(
    decorationAfterClose.inlineCount > 0,
    `inline marker should remain after closing widget: ${JSON.stringify(decorationAfterClose)}`
  );
  assertCheck(
    (decorationAfterClose.widgetTexts || []).length === 0,
    `error widget did not close: ${JSON.stringify(decorationAfterClose)}`
  );
  return { decorationAfterClose, closeButtonPlacement };
}

async function runCase(page, item, summary) {
  await setWebSQLEditorValue(page, item.editorSql || item.sql);
  let selected = null;
  if (item.selectedSql) {
    selected = await selectWebSQLEditorText(page, item.selectedSql);
  }
  const execution = await waitForExecutionResponse(page, async () => {
    await clickExecute(page);
  }, item.sql);
  const body = execution.json || {};
  const data = body.data || {};
  const loc = data.error_location || {};
  const marker = Array.isArray(data.markers) ? data.markers[0] || {} : {};
  assertCheck(body.success === false, `${item.name}: expected failed response`);
  assertCheck(
    String(execution.payload.service || "") === "webshell.websql_execute",
    `${item.name}: request did not include service`
  );
  assertCheck(
    String(execution.payload.driver || "") === "sqlite",
    `${item.name}: request did not use sqlite`
  );
  if (item.expectedSelectionStartLine) {
    assertCheck(
      Number(execution.payload.selection_start_line) === item.expectedSelectionStartLine &&
        Number(execution.payload.selection_start_column) === item.expectedSelectionStartColumn,
      `${item.name}: selection offset was not sent correctly: ${JSON.stringify(execution.payload)}`
    );
  }
  assertCheck(
    String(loc.kind || "") === item.expectedKind,
    `${item.name}: expected kind ${item.expectedKind}, got ${loc.kind}`
  );
  assertCheck(
    Number(loc.line) === item.expectedLine &&
      Number(loc.column) === item.expectedColumn,
    `${item.name}: expected location ${item.expectedLine}:${item.expectedColumn}, got ${loc.line}:${loc.column}`
  );
  assertCheck(
    String(loc.token || "") === item.expectedToken,
    `${item.name}: expected token ${item.expectedToken}, got ${loc.token}`
  );
  assertCheck(
    String(loc.direction_label || "") === item.expectedDirectionLabel,
    `${item.name}: expected direction ${item.expectedDirectionLabel}, got ${loc.direction_label}`
  );
  assertCheck(
    Number(marker.startLineNumber) === item.expectedLine &&
      Number(marker.startColumn) === item.expectedColumn,
    `${item.name}: response marker location mismatch`
  );
  assertCheck(
    String(marker.raw_message || "").includes(item.expectedRawText),
    `${item.name}: response marker raw_message missing ${item.expectedRawText}; marker=${JSON.stringify(marker)}`
  );
  const editorMarker = await waitForMarker(page, item);
  const editorDecoration = await waitForEditorDecoration(page, item);
  const errorPanelText = await readVisibleErrorPanelText(page);
  for (const expectedText of item.expectedPanelText || []) {
    assertCheck(
      errorPanelText.includes(expectedText),
      `${item.name}: error panel missing ${expectedText}; text=${errorPanelText}`
    );
  }
  const screenshotPath = path.join(
    OUT_DIR,
    `websql-${item.name}-error-marker-check.png`
  );
  await page.screenshot({ path: screenshotPath, fullPage: true });
  const { decorationAfterClose, closeButtonPlacement } =
    await closeVisibleEditorErrorWidget(page);
  summary.cases.push({
    name: item.name,
    sql: item.sql,
    response: {
      success: body.success,
      msg: body.msg,
      errorLocation: loc,
      marker,
    },
    request: {
      service: execution.payload.service,
      driver: execution.payload.driver,
      sqlitePath: execution.payload.sqlite_path,
      sql: execution.payload.sql,
      selectionStartLine: execution.payload.selection_start_line,
      selectionStartColumn: execution.payload.selection_start_column,
    },
    selected,
    editorMarker: editorMarker.hit,
    editorMarkers: editorMarker.markers,
    editorDecoration,
    decorationAfterClose,
    closeButtonPlacement,
    errorPanelText,
    screenshot: screenshotPath,
  });
}

async function main() {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const reportPath = path.join(OUT_DIR, "websql-sqlite-error-marker-check.json");
  const summary = {
    url: TARGET_URL,
    cases: [],
    consoleErrors: [],
    pageErrors: [],
    requestFailed: [],
    report: reportPath,
  };

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1600, height: 960 } });

  page.on("console", (msg) => {
    if (msg.type() === "error") {
      summary.consoleErrors.push(msg.text());
    }
  });
  page.on("pageerror", (error) => {
    summary.pageErrors.push(String(error));
  });
  page.on("requestfailed", (request) => {
    summary.requestFailed.push({
      url: request.url(),
      method: request.method(),
      errorText: request.failure()?.errorText || "",
    });
  });

  try {
    await gotoPage(page, TARGET_URL);
    await clearWebSQLLocalState(page);
    await gotoPage(page, TARGET_URL);
    await page.locator(".websql-lowcode").first().waitFor({
      state: "visible",
      timeout: 30000,
    });
    await page.locator(".websql-lowcode .monaco-editor").first().waitFor({
      state: "visible",
      timeout: 45000,
    });

    for (const item of CASES) {
      await runCase(page, item, summary);
    }

    assertCheck(
      summary.consoleErrors.length === 0,
      `console errors: ${summary.consoleErrors.join(" | ")}`
    );
    assertCheck(
      summary.pageErrors.length === 0,
      `page errors: ${summary.pageErrors.join(" | ")}`
    );
    assertCheck(
      summary.requestFailed.length === 0,
      `request failed: ${JSON.stringify(summary.requestFailed)}`
    );
  } finally {
    fs.writeFileSync(reportPath, JSON.stringify(summary, null, 2));
    await page.close().catch(() => undefined);
    await browser.close().catch(() => undefined);
  }
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
