#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');
const { chromium } = require('playwright');

const PAGE_URL = process.env.WEBSHELL_EDITOR_POOL_PAGE_URL || 'http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool';
const API_URL = process.env.WEBSHELL_EDITOR_POOL_API_URL || 'http://127.0.0.1:8015/template_data/data';
const OUT_DIR = process.env.WEBSHELL_EDITOR_POOL_OUTPUT_DIR || '/data/project/sport/test/lowcode-page/results/latest/http-proxy-validation';

const REQUIRED_TYPE_LABELS = ['Go 项目', 'Python 项目', '前端项目', 'Node 项目', 'Java 项目'];
const FRONTEND_CODES = ['collect-ui', 'frontend', 'sport-ui'];

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function runCurl(service, data) {
  const payload = JSON.stringify(Object.assign({ service }, data || {}));
  const res = spawnSync('curl', [
    '--noproxy',
    '*',
    '-sS',
    '-m',
    '40',
    `${API_URL}?service=${service}`,
    '-H',
    'Content-Type: application/json',
    '--data',
    payload,
  ], { encoding: 'utf8' });
  if (res.status !== 0) {
    throw new Error(res.stderr || `curl failed: ${service}`);
  }
  let out = {};
  try {
    out = JSON.parse(String(res.stdout || '{}'));
  } catch (error) {
    throw new Error(`parse response failed (${service}): ${error.message}`);
  }
  if (!out || String(out.code || '') !== '0' || out.success === false) {
    throw new Error(`${service} failed: ${out?.msg || 'unknown error'}`);
  }
  return out;
}

function escapeRegExp(input) {
  return String(input || '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

async function getTopVisibleModal(page) {
  const modals = page.locator('.ant-modal-wrap:visible .ant-modal');
  const count = await modals.count();
  if (!count) {
    throw new Error('visible modal not found');
  }
  return modals.nth(count - 1);
}

async function selectProjectType(dialog, label) {
  const item = dialog.locator('.ant-form-item').filter({ hasText: '项目类型' }).first();
  await item.waitFor({ state: 'visible', timeout: 15000 });
  const selector = item.locator('.ant-select').first();
  await selector.click();
  const option = dialog.page().locator('.ant-select-dropdown:visible .ant-select-item-option-content')
    .filter({ hasText: new RegExp(`^${escapeRegExp(label)}$`) }).first();
  await option.waitFor({ state: 'visible', timeout: 10000 });
  await option.click();
  await sleep(200);
}

async function listProjectTypeOptions(dialog) {
  const item = dialog.locator('.ant-form-item').filter({ hasText: '项目类型' }).first();
  await item.waitFor({ state: 'visible', timeout: 15000 });
  await item.locator('.ant-select').first().click();
  await sleep(180);
  const options = await dialog.page().locator('.ant-select-dropdown:visible .ant-select-item-option-content')
    .allTextContents();
  await dialog.page().keyboard.press('Escape');
  await sleep(120);
  return options.map((itemText) => String(itemText || '').trim()).filter(Boolean);
}

async function isFieldVisible(dialog, label) {
  return dialog.evaluate((root, fieldLabel) => {
    const items = Array.from(root.querySelectorAll('.ant-form-item'));
    for (const item of items) {
      const labelEl = item.querySelector('.ant-form-item-label label, .ant-form-item-label');
      const text = String(labelEl?.textContent || '').trim();
      if (text !== String(fieldLabel || '').trim()) {
        continue;
      }
      const style = window.getComputedStyle(item);
      const rect = item.getBoundingClientRect();
      if (style.display === 'none' || style.visibility === 'hidden') {
        return false;
      }
      return rect.height > 0 && rect.width > 0;
    }
    return false;
  }, label);
}

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  const summary = {
    pass: false,
    pageUrl: PAGE_URL,
    apiUrl: API_URL,
    startedAt: new Date().toISOString(),
    steps: {
      apiProjectTypeCheck: false,
      openProjectManage: false,
      openProjectDialog: false,
      verifyTypeOptions: false,
      verifyPythonVisibility: false,
      verifyGoVisibility: false,
      verifyFrontendVisibility: false,
    },
    details: {
      projectTypeByCode: {},
      dropdownOptions: [],
      pythonMode: {},
      goMode: {},
      frontendMode: {},
    },
    screenshot: '',
    error: '',
  };

  let browser;
  try {
    const projectRes = runCurl('webshell.workspace_project_query', { pagination: false });
    const rows = Array.isArray(projectRes.data) ? projectRes.data : [];
    for (const row of rows) {
      summary.details.projectTypeByCode[String(row.project_code || '')] = String(row.project_type || '');
    }
    for (const code of FRONTEND_CODES) {
      if (summary.details.projectTypeByCode[code] !== 'frontend') {
        throw new Error(`project_type mismatch: ${code} -> ${summary.details.projectTypeByCode[code] || 'N/A'}`);
      }
    }
    summary.steps.apiProjectTypeCheck = true;

    browser = await chromium.launch({ headless: true });
    const page = await browser.newPage({ viewport: { width: 1600, height: 960 } });
    page.on('dialog', async (dialog) => {
      await dialog.dismiss().catch(() => {});
    });

    await page.goto(PAGE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await page.waitForTimeout(1600);

    const manageBtn = page.getByRole('button', { name: '项目管理' }).first();
    await manageBtn.waitFor({ state: 'visible', timeout: 25000 });
    await manageBtn.click();

    const manageDialog = await getTopVisibleModal(page);
    await manageDialog.locator('.ant-modal-title').filter({ hasText: '项目管理' }).first()
      .waitFor({ state: 'visible', timeout: 10000 });
    summary.steps.openProjectManage = true;

    const addBtn = manageDialog.getByRole('button', { name: '新增项目' }).first();
    await addBtn.waitFor({ state: 'visible', timeout: 12000 });
    await addBtn.click();

    const projectDialog = await getTopVisibleModal(page);
    await projectDialog.locator('.ant-modal-title').filter({ hasText: '新增项目' }).first()
      .waitFor({ state: 'visible', timeout: 12000 });
    summary.steps.openProjectDialog = true;

    const options = await listProjectTypeOptions(projectDialog);
    summary.details.dropdownOptions = options;
    for (const requiredLabel of REQUIRED_TYPE_LABELS) {
      if (!options.includes(requiredLabel)) {
        throw new Error(`missing dropdown option: ${requiredLabel}`);
      }
    }
    summary.steps.verifyTypeOptions = true;

    await selectProjectType(projectDialog, 'Python 项目');
    const pythonFieldVisible = await isFieldVisible(projectDialog, 'Python 安装包路径');
    const goFieldVisibleInPython = await isFieldVisible(projectDialog, 'Go collect源码路径');
    summary.details.pythonMode = {
      pythonPkgPathVisible: pythonFieldVisible,
      goCollectPathVisible: goFieldVisibleInPython,
    };
    if (!pythonFieldVisible || goFieldVisibleInPython) {
      throw new Error('python mode field visibility mismatch');
    }
    summary.steps.verifyPythonVisibility = true;

    await selectProjectType(projectDialog, 'Go 项目');
    const goFieldVisible = await isFieldVisible(projectDialog, 'Go collect源码路径');
    const pythonFieldVisibleInGo = await isFieldVisible(projectDialog, 'Python 安装包路径');
    summary.details.goMode = {
      goCollectPathVisible: goFieldVisible,
      pythonPkgPathVisible: pythonFieldVisibleInGo,
    };
    if (!goFieldVisible || pythonFieldVisibleInGo) {
      throw new Error('go mode field visibility mismatch');
    }
    summary.steps.verifyGoVisibility = true;

    await selectProjectType(projectDialog, '前端项目');
    const goFieldVisibleInFrontend = await isFieldVisible(projectDialog, 'Go collect源码路径');
    const pythonFieldVisibleInFrontend = await isFieldVisible(projectDialog, 'Python 安装包路径');
    summary.details.frontendMode = {
      goCollectPathVisible: goFieldVisibleInFrontend,
      pythonPkgPathVisible: pythonFieldVisibleInFrontend,
    };
    if (goFieldVisibleInFrontend || pythonFieldVisibleInFrontend) {
      throw new Error('frontend mode field visibility mismatch');
    }
    summary.steps.verifyFrontendVisibility = true;

    summary.screenshot = path.join(OUT_DIR, 'webshell-editor-pool-project-type-ui-check.png');
    await page.screenshot({ path: summary.screenshot, fullPage: true });
    summary.pass = true;
  } catch (error) {
    summary.error = String(error?.message || error);
  } finally {
    if (browser) {
      await browser.close().catch(() => {});
    }
    summary.endedAt = new Date().toISOString();
    const outFile = path.join(OUT_DIR, 'webshell-editor-pool-project-type-ui-check.json');
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
