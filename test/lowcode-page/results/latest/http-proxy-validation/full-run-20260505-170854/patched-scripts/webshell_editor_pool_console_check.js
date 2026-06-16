#!/usr/bin/env node

const fs = require("fs");
const path = require("path");

async function main() {
  let playwright;
  try {
    playwright = require("playwright");
  } catch (error) {
    console.error("playwright is required to run this script");
    process.exit(1);
  }

  const { chromium } = playwright;
  const baseUrl = process.env.WEBSHELL_EDITOR_POOL_BASE_URL || "http://192.168.232.130:8015";
  const pageUrl =
    process.env.WEBSHELL_EDITOR_POOL_PAGE_URL ||
    `${baseUrl}/collect-ui#/collect-ui/framework/webshell-editor-pool`;
  const outputDir =
    process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR ||
    path.join(process.cwd(), "test/lowcode-page/results/latest/http-proxy-validation");
  const resultPath = path.join(outputDir, "webshell-editor-pool-http-console-result.json");

  const screenshotDocPath = path.join(outputDir, "webshell-editor-pool-http-doc.png");
  const screenshotCollapsedPath = path.join(outputDir, "webshell-editor-pool-http-console-collapsed.png");
  const screenshotExpandedPath = path.join(outputDir, "webshell-editor-pool-http-console-expanded.png");
  const screenshotCollapsedAgainPath = path.join(outputDir, "webshell-editor-pool-http-console-collapsed-again.png");
  const bodyPath = path.join(outputDir, "webshell-editor-pool-http-console-body.txt");

  fs.mkdirSync(outputDir, { recursive: true });

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1680, height: 980 } });
  const page = await context.newPage();

  const consoleErrors = [];
  const pageErrors = [];
  const failedRequests = [];
  const actionTrace = [];

  page.on("console", (msg) => {
    if (msg.type() === "error") {
      consoleErrors.push(msg.text());
    }
  });
  page.on("pageerror", (err) => {
    pageErrors.push(String(err));
  });
  page.on("requestfailed", (req) => {
    failedRequests.push(`${req.method()} ${req.url()} => ${req.failure()?.errorText || "failed"}`);
  });

  const clickIfVisible = async (locator, label) => {
    const count = await locator.count();
    if (!count) return false;
    const first = locator.first();
    const visible = await first.isVisible().catch(() => false);
    if (!visible) return false;
    await first.click();
    actionTrace.push(`clicked: ${label}`);
    return true;
  };

  try {
    await page.goto(pageUrl, { waitUntil: "networkidle", timeout: 45000 });
    await page.waitForTimeout(2400);

    const httpTab = page.getByText("HTTP目录").first();
    if (await httpTab.isVisible().catch(() => false)) {
      await httpTab.click();
      actionTrace.push("clicked: HTTP目录");
      await page.waitForTimeout(1000);
    }

    let openedGroup = "";
    const preferredGroups = ["SSH", "test2", "文档管理", "ldap", "hrm"];
    for (const groupName of preferredGroups) {
      const groupTreeNode = page.locator(".ant-tree-treenode", {
        has: page.getByText(new RegExp(`^${groupName}$`)),
      }).first();
      if (await groupTreeNode.isVisible().catch(() => false)) {
        const switcher = groupTreeNode.locator(".ant-tree-switcher").first();
        if (await switcher.isVisible().catch(() => false)) {
          await switcher.click();
          await page.waitForTimeout(700);
        } else {
          await groupTreeNode.click();
          await page.waitForTimeout(700);
        }
        openedGroup = groupName;
        actionTrace.push(`opened group: ${groupName}`);
        break;
      }
    }

    // Try to open a concrete HTTP service entry.
    let clickedService = "";
    const preferredServiceRegex = [
      /get_env_detail_service/,
      /sync_doc/,
      /ldap_search/,
      /my_env_tree/,
      /test\(tset\)/,
    ];
    for (const matcher of preferredServiceRegex) {
      const serviceNode = page.getByText(matcher).first();
      if (await serviceNode.isVisible().catch(() => false)) {
        await serviceNode.click();
        await page.waitForTimeout(900);
        clickedService = await serviceNode.textContent();
        actionTrace.push(`opened service: ${clickedService || matcher.toString()}`);
        break;
      }
    }

    if (!clickedService) {
      const fallbackService = page.locator(".ant-tree-title").filter({ hasText: "(" }).first();
      if (await fallbackService.isVisible().catch(() => false)) {
        clickedService = (await fallbackService.textContent()) || "";
        await fallbackService.click();
        await page.waitForTimeout(900);
        actionTrace.push(`opened fallback service: ${clickedService}`);
      }
    }

    if (!clickedService) {
      throw new Error("failed to locate and open an HTTP service node");
    }

    await page.screenshot({ path: screenshotDocPath, fullPage: true });

    let clickedConsole = false;
    if (await clickIfVisible(page.getByRole("button", { name: "打开控制台" }).first(), "打开控制台")) {
      clickedConsole = true;
    } else if (await clickIfVisible(page.getByRole("button", { name: /^控制台$/ }).first(), "控制台")) {
      clickedConsole = true;
    }
    if (!clickedConsole) {
      throw new Error("failed to click console entry button");
    }
    await page.waitForSelector(".workspace-http-console-root", { timeout: 20000 });
    await page.waitForTimeout(900);

    const hasConsoleRoot = await page.locator(".workspace-http-console-root").first().isVisible().catch(() => false);
    const hasSendButton = await page.locator(".workspace-http-console-send").first().isVisible().catch(() => false);
    const hasUrlInput = await page.locator(".workspace-http-console-url").first().isVisible().catch(() => false);
    const hasMethodSelect = await page.locator(".workspace-http-console-method").first().isVisible().catch(() => false);
    const hasHeadersTitle = await page.getByText("Headers").first().isVisible().catch(() => false);

    const requestHeadersPanel = page.getByText("REQUEST HEADERS").first();
    const headersExpandedBefore = await requestHeadersPanel.isVisible().catch(() => false);
    await page.screenshot({ path: screenshotCollapsedPath, fullPage: true });

    // Expand
    await page.getByText("Headers").first().click();
    actionTrace.push("clicked: Headers expand");
    await page.waitForTimeout(700);
    const headersExpandedAfterExpand = await requestHeadersPanel.isVisible().catch(() => false);
    await page.screenshot({ path: screenshotExpandedPath, fullPage: true });

    // Collapse again
    await page.getByText("Headers").first().click();
    actionTrace.push("clicked: Headers collapse");
    await page.waitForTimeout(700);
    const headersExpandedAfterCollapse = await requestHeadersPanel.isVisible().catch(() => false);
    await page.screenshot({ path: screenshotCollapsedAgainPath, fullPage: true });

    // Optional send request click for runtime check.
    await clickIfVisible(page.locator(".workspace-http-console-send"), "Send");
    await page.waitForTimeout(1500);

    const headerTip = (await page.locator(".workspace-http-console-header-tip").first().textContent().catch(() => "")) || "";
    const methodClass = (await page.locator(".workspace-http-console-method").first().getAttribute("class").catch(() => "")) || "";

    const bodyText = await page.locator("body").textContent();
    fs.writeFileSync(bodyPath, bodyText || "");

    const detail = {
      pageUrl,
      finalUrl: page.url(),
      openedGroup,
      clickedService: clickedService || null,
      hasConsoleRoot,
      hasSendButton,
      hasUrlInput,
      hasMethodSelect,
      hasHeadersTitle,
      headerCollapse: {
        defaultCollapsed: !headersExpandedBefore,
        expandedAfterClick: headersExpandedAfterExpand,
        collapsedAfterSecondClick: !headersExpandedAfterCollapse,
        headerTip,
      },
      postmanStyleSignals: {
        methodClass,
        hasRequestBar: await page.locator(".workspace-http-console-request-bar").first().isVisible().catch(() => false),
        hasPanelCard: await page.locator(".workspace-http-console-panel").first().isVisible().catch(() => false),
        hasOrangeSendButton: await page
          .locator(".workspace-http-console-send")
          .evaluate((el) => window.getComputedStyle(el).backgroundImage || "")
          .catch(() => ""),
      },
      screenshots: {
        doc: screenshotDocPath,
        consoleCollapsed: screenshotCollapsedPath,
        consoleExpanded: screenshotExpandedPath,
        consoleCollapsedAgain: screenshotCollapsedAgainPath,
      },
      actionTrace,
      consoleErrors,
      pageErrors,
      failedRequests,
    };

    fs.writeFileSync(resultPath, JSON.stringify(detail, null, 2));
    console.log(JSON.stringify(detail, null, 2));
  } finally {
    await page.close();
    await context.close();
    await browser.close();
  }
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
