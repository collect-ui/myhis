#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');

let playwright;
try {
  playwright = require('playwright');
} catch (_error) {
  try {
    playwright = require('/data/project/sport-ui/node_modules/playwright');
  } catch (error) {
    console.error('playwright is required to run this script');
    console.error(String(error && error.message ? error.message : error));
    process.exit(1);
  }
}

const { chromium } = playwright;

const PAGE_URL =
  process.env.WEBSHELL_EDITOR_POOL_PAGE_URL ||
  'http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool';
const API_URL =
  process.env.WEBSHELL_EDITOR_POOL_API_URL ||
  'http://127.0.0.1:8015/template_data/data';
const OUT_DIR =
  process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR ||
  '/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation';

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function runCurl(service, data) {
  const payload = JSON.stringify(Object.assign({ service }, data || {}));
  const res = spawnSync(
    'curl',
    [
      '--noproxy',
      '*',
      '-s',
      '-m',
      '30',
      `${API_URL}?service=${service}`,
      '-H',
      'Content-Type: application/json',
      '--data',
      payload,
    ],
    { encoding: 'utf8' },
  );
  if (res.status !== 0) {
    throw new Error(res.stderr || `curl failed: ${service}`);
  }
  return JSON.parse(String(res.stdout || '{}'));
}

function pickHomeProjects() {
  const res = runCurl('webshell.workspace_project_query', { page: 1, size: 200 });
  const list = Array.isArray(res.data) ? res.data : [];
  return list
    .filter((item) => String(item.show_home || '') === '1')
    .sort((a, b) => Number(a.order_id || 0) - Number(b.order_id || 0));
}

function findCaseTarget(projects, oneCase) {
  for (const p of projects) {
    for (const kw of oneCase.keywords) {
      const res = runCurl('webshell.workspace_file_content_search', {
        project_code: p.project_code,
        keyword: kw,
        include_glob: oneCase.includeGlob,
        max_results: 5,
      });
      const items = Array.isArray(res && res.data && res.data.items)
        ? res.data.items
        : [];
      if (String(res.code || '') === '0' && items.length > 0) {
        return {
          project_code: p.project_code,
          project_name: p.project_name,
          keyword: kw,
          file_path: String(items[0].file_path || ''),
          line_no: Number(items[0].line_no || 1),
        };
      }
    }
  }
  return null;
}

async function chooseIncludeOption(page, optionText) {
  const select = page
    .locator('.ant-modal')
    .filter({ hasText: '内容搜索' })
    .first()
    .locator('.ant-select')
    .first();
  await select.click();
  await sleep(200);
  await page
    .locator('.ant-select-dropdown .ant-select-item-option')
    .filter({ hasText: optionText })
    .first()
    .click();
  await sleep(200);
}

async function openBySearch(page, target, oneCase) {
  const projectBtn = page.getByRole('button', { name: target.project_name }).first();
  if (await projectBtn.isVisible().catch(() => false)) {
    await projectBtn.click();
    await sleep(700);
  }

  await page.locator('button[title="内容搜索"]').first().click();
  await sleep(300);

  const modal = page.locator('.ant-modal').filter({ hasText: '内容搜索' }).first();
  await modal.waitFor({ state: 'visible', timeout: 10000 });

  const keywordInput = modal
    .locator('input[placeholder="输入关键字(至少2个字符)"]')
    .first();
  await keywordInput.fill('');
  await keywordInput.fill(target.keyword);

  await chooseIncludeOption(page, oneCase.optionText);
  await modal.getByRole('button', { name: '搜索' }).first().click();
  await sleep(1000);

  const firstLine = page.locator('text=/第\\d+行/').first();
  await firstLine.waitFor({ state: 'visible', timeout: 15000 });
  await firstLine.click();
  await sleep(1400);
}

async function getEditorSnapshot(page) {
  return page.evaluate(() => {
    const monaco = window && window.monaco && window.monaco.editor;
    if (!monaco || typeof monaco.getEditors !== 'function') {
      return null;
    }
    const editors = monaco.getEditors() || [];
    let best = null;
    let bestArea = 0;
    for (const ed of editors) {
      try {
        if (!ed || typeof ed.getModel !== 'function') continue;
        const model = ed.getModel();
        if (!model) continue;
        const node = ed.getContainerDomNode && ed.getContainerDomNode();
        if (!node || !node.isConnected) continue;
        const rect = node.getBoundingClientRect();
        if (rect.width < 120 || rect.height < 80) continue;
        const st = getComputedStyle(node);
        if (st.display === 'none' || st.visibility === 'hidden') continue;
        const area = rect.width * rect.height;
        if (area > bestArea) {
          bestArea = area;
          best = { ed, model };
        }
      } catch (_error) {
        // ignore
      }
    }
    if (!best) return null;
    return {
      uri: decodeURIComponent(String((best.model && best.model.uri) || '')),
      language: String(
        (best.model && best.model.getLanguageId && best.model.getLanguageId()) ||
          '',
      ),
      value: String((best.ed && best.ed.getValue && best.ed.getValue()) || ''),
    };
  });
}

async function setEditorValue(page, nextValue) {
  return page.evaluate((nextValueInner) => {
    const monaco = window && window.monaco && window.monaco.editor;
    if (!monaco || typeof monaco.getEditors !== 'function') {
      return false;
    }
    const editors = monaco.getEditors() || [];
    let best = null;
    let bestArea = 0;
    for (const ed of editors) {
      try {
        const model = ed && ed.getModel && ed.getModel();
        if (!model) continue;
        const node = ed.getContainerDomNode && ed.getContainerDomNode();
        if (!node || !node.isConnected) continue;
        const rect = node.getBoundingClientRect();
        if (rect.width < 120 || rect.height < 80) continue;
        const st = getComputedStyle(node);
        if (st.display === 'none' || st.visibility === 'hidden') continue;
        const area = rect.width * rect.height;
        if (area > bestArea) {
          bestArea = area;
          best = ed;
        }
      } catch (_error) {
        // ignore
      }
    }
    if (!best) return false;
    try {
      best.setValue(String(nextValueInner || ''));
      return true;
    } catch (_error) {
      return false;
    }
  }, nextValue);
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const ts = new Date().toISOString().replace(/[:.]/g, '-');

  const cases = [
    {
      id: 'sql',
      includeGlob: '*.sql',
      optionText: 'SQL (*.sql)',
      keywords: ['select', 'from', 'where'],
      sample: "select  id,name  from demo_table where id=1 and name='x' order   by id desc",
      verify: (v) => /\n/.test(v) && /from/i.test(v),
    },
    {
      id: 'python',
      includeGlob: '*.py',
      optionText: 'Python (*.py)',
      keywords: ['import', 'def', 'class', 'return'],
      sample: "def foo():\n\tprint('x')  \n\tif True:\n\t\treturn 1",
      verify: (v) => !/\t/.test(v) && /\n$/.test(v),
    },
    {
      id: 'yaml',
      includeGlob: '*.yaml,*.yml',
      optionText: 'YAML (*.yaml,*.yml)',
      keywords: ['module', 'service', 'name', 'key'],
      sample: "root:\n\tchild: value  \n\tlist:\n\t\t- a\n\t\t- b  ",
      verify: (v) => !/\t/.test(v) && /\n$/.test(v),
    },
  ];

  const projects = pickHomeProjects();
  const targets = {};
  for (const oneCase of cases) {
    targets[oneCase.id] = findCaseTarget(projects, oneCase);
  }

  const summary = {
    pageUrl: PAGE_URL,
    apiUrl: API_URL,
    startedAt: new Date().toISOString(),
    projects: projects.map((p) => ({ code: p.project_code, name: p.project_name })),
    targets,
    cases: {},
    consoleErrors: [],
    pageErrors: [],
    failedRequests: [],
    pass: false,
  };

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1720, height: 980 } });
  page.setDefaultTimeout(20000);
  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      summary.consoleErrors.push(msg.text());
    }
  });
  page.on('pageerror', (err) => summary.pageErrors.push(String(err)));
  page.on('requestfailed', (req) => {
    summary.failedRequests.push(
      `${req.method()} ${req.url()} => ${req.failure()?.errorText || 'failed'}`,
    );
  });

  try {
    await page.goto(PAGE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await sleep(6000);

    for (const oneCase of cases) {
      const target = targets[oneCase.id];
      if (!target) {
        summary.cases[oneCase.id] = {
          ok: false,
          reason: 'no_target_file_found',
        };
        continue;
      }

      await openBySearch(page, target, oneCase);
      const opened = await getEditorSnapshot(page);

      const setOk = await setEditorValue(page, oneCase.sample);
      await sleep(200);
      const beforeFormat = await getEditorSnapshot(page);

      await page.getByRole('button', { name: '格式化' }).first().click();
      await sleep(1200);
      const afterFormat = await getEditorSnapshot(page);

      const screenshot = path.join(
        OUT_DIR,
        `webshell-editor-pool-format-${oneCase.id}-${ts}.png`,
      );
      await page.screenshot({ path: screenshot, fullPage: true });

      const changed =
        !!beforeFormat && !!afterFormat && beforeFormat.value !== afterFormat.value;
      const verifyPass =
        !!afterFormat && oneCase.verify(String(afterFormat.value || ''));

      summary.cases[oneCase.id] = {
        ok: Boolean(opened && setOk && changed && verifyPass),
        target,
        openedUri: opened?.uri || '',
        openedLanguage: opened?.language || '',
        setOk,
        before: beforeFormat?.value || '',
        after: afterFormat?.value || '',
        changed,
        verifyPass,
        screenshot,
      };
    }
  } finally {
    await browser.close();
  }

  summary.pass =
    Object.values(summary.cases).every((item) => item && item.ok === true) &&
    summary.pageErrors.length === 0;

  const outFile = path.join(
    OUT_DIR,
    `webshell-editor-pool-format-multi-check-${ts}.json`,
  );
  fs.writeFileSync(outFile, `${JSON.stringify(summary, null, 2)}\n`, 'utf8');

  console.log(
    JSON.stringify(
      {
        pass: summary.pass,
        outFile,
        cases: summary.cases,
        pageErrors: summary.pageErrors.length,
        consoleErrors: summary.consoleErrors.length,
      },
      null,
      2,
    ),
  );

  process.exit(summary.pass ? 0 : 1);
})();
