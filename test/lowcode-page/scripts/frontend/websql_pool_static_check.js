#!/usr/bin/env node

const fs = require("fs");
const path = require("path");

const ROOT = process.env.WEBSQL_STATIC_ROOT || "/data/project/sport";
const PAGE_FILE =
  process.env.WEBSQL_POOL_JSON ||
  path.join(ROOT, "collect/frontend/page_data/data/server/websql_pool.json");
const OUT_DIR =
  process.env.WEBSQL_OUTPUT_DIR ||
  path.join(ROOT, "test/lowcode-page/results/latest/http-proxy-validation");

function assertCheck(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function readPage() {
  return JSON.parse(fs.readFileSync(PAGE_FILE, "utf8"));
}

function walk(value, visit) {
  if (!value || typeof value !== "object") {
    return;
  }
  visit(value);
  if (Array.isArray(value)) {
    value.forEach((item) => walk(item, visit));
    return;
  }
  Object.values(value).forEach((item) => walk(item, visit));
}

function hasClass(node, className) {
  return String(node.className || "").includes(className);
}

function findByClass(root, className) {
  const matches = [];
  walk(root, (node) => {
    if (hasClass(node, className)) {
      matches.push(node);
    }
  });
  return matches;
}

function findByLabel(root, label) {
  const matches = [];
  walk(root, (node) => {
    if (node.label === label) {
      matches.push(node);
    }
  });
  return matches;
}

function findButtonByText(root, text) {
  const matches = [];
  walk(root, (node) => {
    if (node.tag === "button" && node.children === text) {
      matches.push(node);
    }
  });
  return matches;
}

function stringContains(root, text) {
  return JSON.stringify(root).includes(text);
}

function paneClass(index) {
  if (index === 1) {
    return "websql-primary-pane";
  }
  return `websql-pane-${index}`;
}

function paneStorePrefix(index) {
  if (index === 1) {
    return "websql";
  }
  return `websqlPane${index}`;
}

function run() {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const page = readPage();
  const summary = {
    pageFile: PAGE_FILE,
    checks: {},
    panes: [],
  };

  assertCheck(page.initStore && page.initStore.websqlPaneCount === 1, "default websqlPaneCount should be 1");
  assertCheck(page.initStore.websqlActivePaneIndex === 1, "default websqlActivePaneIndex should be 1");
  summary.checks.defaultSinglePaneStore = true;

  assertCheck(findByClass(page, "websql-standalone-main-split").length >= 1, "standalone single-pane split is missing");
  assertCheck(findByClass(page, "websql-workbench-split").length >= 1, "multi-pane split container is missing");
  summary.checks.splitContainers = true;

  for (let index = 1; index <= 4; index += 1) {
    const className = paneClass(index);
    const panes = findByClass(page, className);
    assertCheck(panes.length >= 1, `pane ${index} container ${className} is missing`);
    const pane = panes[0];
    const prefix = paneStorePrefix(index);
    const text = JSON.stringify(pane);
    const sqlTabsKey = index === 1 ? "websqlSqlTabs" : `${prefix}SqlTabs`;
    const activeTabKey = index === 1 ? "websqlActiveSqlTabKey" : `${prefix}ActiveSqlTabKey`;
    const sqlTextKey = index === 1 ? "websqlSqlText" : `${prefix}SqlText`;
    const resultRowsKey = index === 1 ? "websqlResultRows" : `${prefix}ResultRows`;

    assertCheck(findByClass(pane, "websql-pane-toolbar").length >= 1, `pane ${index} toolbar is missing`);
    assertCheck(findByClass(pane, "workspace-websql-sql-tabs").length >= 1, `pane ${index} sql tabs are missing`);
    assertCheck(findByClass(pane, "websql-add-sql-btn").length >= 1, `pane ${index} add SQL button is missing`);
    assertCheck(findByClass(pane, "websql-execute-btn").length >= 1, `pane ${index} execute button is missing`);
    assertCheck(findByClass(pane, "websql-editor-body").length >= 1, `pane ${index} editor body is missing`);
    assertCheck(findByClass(pane, "websql-result-tabs").length >= 1, `pane ${index} result tabs are missing`);
    assertCheck(text.includes(sqlTabsKey), `pane ${index} does not bind ${sqlTabsKey}`);
    assertCheck(text.includes(activeTabKey), `pane ${index} does not bind ${activeTabKey}`);
    assertCheck(text.includes(sqlTextKey), `pane ${index} does not bind ${sqlTextKey}`);
    assertCheck(text.includes(resultRowsKey), `pane ${index} does not bind ${resultRowsKey}`);

    summary.panes.push({
      index,
      className,
      sqlTabsKey,
      activeTabKey,
      sqlTextKey,
      resultRowsKey,
    });
  }
  summary.checks.eachPaneIsFullWorkbench = true;

  const horizontalMenus = findByLabel(page, "左右分割");
  const verticalMenus = findByLabel(page, "上下分割");
  assertCheck(horizontalMenus.length >= 1, "left/right split menu is missing");
  assertCheck(verticalMenus.length >= 1, "top/bottom split menu is missing");
  assertCheck(stringContains(horizontalMenus, "Math.min(4") || stringContains(page, "Math.min(4"), "split action should cap pane count at 4");
  assertCheck(stringContains(page, "websqlPane2SqlTabs"), "split action does not initialize pane 2 tabs");
  assertCheck(stringContains(page, "websqlPane3SqlTabs"), "split action does not initialize pane 3 tabs");
  assertCheck(stringContains(page, "websqlPane4SqlTabs"), "split action does not initialize pane 4 tabs");
  summary.checks.recursiveSplitActions = true;

  assertCheck(findButtonByText(page, "Ping").length === 0, "operation toolbar should not expose Ping buttons");
  assertCheck(findButtonByText(page, "DDL").length === 0, "operation toolbar should not expose DDL buttons");
  summary.checks.noPingOrDdlToolbarButtons = true;

  assertCheck(!stringContains(page, "websql-main > .websql-toolbar"), "page JSON should not rely on a shared toolbar selector");
  summary.checks.noSharedToolbarContract = true;

  summary.ok = true;
  fs.writeFileSync(
    path.join(OUT_DIR, "websql-pool-static-check.json"),
    `${JSON.stringify(summary, null, 2)}\n`
  );
}

try {
  run();
} catch (error) {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  fs.writeFileSync(
    path.join(OUT_DIR, "websql-pool-static-check.json"),
    `${JSON.stringify({ ok: false, pageFile: PAGE_FILE, error: error.message }, null, 2)}\n`
  );
  console.error(error);
  process.exit(1);
}
