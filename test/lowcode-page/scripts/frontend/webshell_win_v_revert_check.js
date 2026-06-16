#!/usr/bin/env node

const fs = require("fs");

let playwright;
try {
  playwright = require("playwright");
} catch (_error) {
  playwright = require("/data/project/sport-ui/node_modules/playwright");
}

const { chromium } = playwright;

const PAGE_URL =
  process.env.WEBSHELL_PAGE_URL ||
  "http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell";
const OUT_DIR =
  process.env.WEBSHELL_OUTPUT_DIR ||
  "/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation";
const TARGET_HOST = process.env.WEBSHELL_TARGET_HOST || "192.168.232.130";

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function countOccurrences(text, marker) {
  return String(text || "").split(marker).length - 1;
}

async function getXtermText(page) {
  return page.locator(".xterm-rows").first().innerText({ timeout: 3000 });
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });

  const marker = `WSH_REVERT_${Date.now()}`;
  const summary = {
    pageUrl: PAGE_URL,
    targetHost: TARGET_HOST,
    marker,
    checks: {
      pageOpened: false,
      targetLoginClicked: false,
      loginConfirmed: false,
      terminalVisible: false,
      noPasteCatcherAttr: false,
      noPasteDebugLog: false,
      xtermTextareaKeptDefault: false,
      markerPrintedOnce: false,
      blankEnterDidNotReplayPreviousInput: false,
    },
    details: {
      httpStatus: "",
      pasteCatcherCount: 0,
      pasteDebugLogLength: 0,
      xtermTextarea: null,
      markerCountAfterCommand: 0,
      markerCountAfterBlankEnter: 0,
      terminalTail: "",
    },
    consoleErrors: [],
    pageErrors: [],
    requestFailed: [],
    screenshots: {
      open: `${OUT_DIR}/webshell-win-v-revert-open.png`,
      terminal: `${OUT_DIR}/webshell-win-v-revert-terminal.png`,
      afterInput: `${OUT_DIR}/webshell-win-v-revert-after-input.png`,
      fail: `${OUT_DIR}/webshell-win-v-revert-fail.png`,
    },
    reportPath: `${OUT_DIR}/webshell-win-v-revert-check.json`,
    pass: false,
    error: "",
  };

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1600, height: 950 } });

  page.on("console", (msg) => {
    if (msg.type() === "error") {
      summary.consoleErrors.push(msg.text());
    }
  });
  page.on("pageerror", (error) => {
    summary.pageErrors.push(String(error));
  });
  page.on("requestfailed", (request) => {
    summary.requestFailed.push(
      `${request.method()} ${request.url()} => ${
        request.failure()?.errorText || "failed"
      }`
    );
  });

  try {
    const response = await page.goto(PAGE_URL, {
      waitUntil: "domcontentloaded",
      timeout: 60000,
    });
    summary.details.httpStatus = response ? String(response.status()) : "";
    summary.checks.pageOpened = response ? response.ok() : false;
    await sleep(5000);
    await page.screenshot({ path: summary.screenshots.open, fullPage: true });

    const targetCard = page
      .locator("div", { hasText: new RegExp(`${TARGET_HOST}（linux）`) })
      .filter({ has: page.getByRole("button", { name: "登录" }) })
      .first();
    const loginButton = targetCard.getByRole("button", { name: "登录" }).first();
    await loginButton.waitFor({ state: "visible", timeout: 20000 });
    await loginButton.click();
    summary.checks.targetLoginClicked = true;

    const confirmButton = page.getByRole("button", { name: "确 定" }).last();
    await confirmButton.waitFor({ state: "visible", timeout: 12000 });
    await confirmButton.click();
    summary.checks.loginConfirmed = true;

    await page.locator(".xterm").first().waitFor({ state: "visible", timeout: 30000 });
    await page
      .locator(".ant-tabs-tab", { hasText: TARGET_HOST })
      .first()
      .waitFor({ state: "visible", timeout: 12000 });
    await sleep(8000);
    await page.screenshot({
      path: summary.screenshots.terminal,
      fullPage: true,
    });
    summary.checks.terminalVisible = true;

    summary.details.pasteCatcherCount = await page
      .locator("[data-webshell-paste-catcher]")
      .count();
    summary.checks.noPasteCatcherAttr =
      summary.details.pasteCatcherCount === 0;

    const pasteDebugLogLength = await page.evaluate(() => {
      const log = window.__webshellPasteDebugLog;
      return Array.isArray(log) ? log.length : 0;
    });
    summary.details.pasteDebugLogLength = pasteDebugLogLength;
    summary.checks.noPasteDebugLog = pasteDebugLogLength === 0;

    summary.details.xtermTextarea = await page
      .locator("textarea.xterm-helper-textarea")
      .first()
      .evaluate((node) => ({
        style: node.getAttribute("style") || "",
        dataPaste: node.getAttribute("data-webshell-paste-catcher"),
        width: node.offsetWidth,
        height: node.offsetHeight,
      }));
    summary.checks.xtermTextareaKeptDefault =
      !summary.details.xtermTextarea.dataPaste &&
      !summary.details.xtermTextarea.style.includes("opacity: 0.01") &&
      !summary.details.xtermTextarea.style.includes("width: 20px");

    await page.locator(".xterm").first().click();
    await page.keyboard.type(`printf '${marker}\\n'`, { delay: 8 });
    await page.keyboard.press("Enter");
    await page.waitForFunction(
      ({ marker }) =>
        (document.querySelector(".xterm-rows")?.innerText || "").split(marker)
          .length -
          1 >=
        2,
      { marker },
      { timeout: 15000 }
    );
    await sleep(800);
    const afterCommandText = await getXtermText(page);
    summary.details.markerCountAfterCommand = countOccurrences(
      afterCommandText,
      marker
    );
    summary.checks.markerPrintedOnce =
      summary.details.markerCountAfterCommand === 2;

    await page.keyboard.press("Enter");
    await sleep(1800);
    const afterBlankEnterText = await getXtermText(page);
    summary.details.markerCountAfterBlankEnter = countOccurrences(
      afterBlankEnterText,
      marker
    );
    summary.details.terminalTail = afterBlankEnterText.slice(-1200);
    summary.checks.blankEnterDidNotReplayPreviousInput =
      summary.details.markerCountAfterBlankEnter ===
      summary.details.markerCountAfterCommand;

    await page.screenshot({
      path: summary.screenshots.afterInput,
      fullPage: true,
    });

    summary.pass =
      Object.values(summary.checks).every(Boolean) &&
      summary.consoleErrors.length === 0 &&
      summary.pageErrors.length === 0 &&
      summary.requestFailed.length === 0;
  } catch (error) {
    summary.error = error && error.stack ? error.stack : String(error);
    await page.screenshot({ path: summary.screenshots.fail, fullPage: true }).catch(() => {});
  } finally {
    fs.writeFileSync(summary.reportPath, JSON.stringify(summary, null, 2));
    await browser.close();
  }

  console.log(JSON.stringify(summary, null, 2));
  if (!summary.pass) {
    process.exit(1);
  }
})();
