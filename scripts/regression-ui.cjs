// Runs against an isolated temporary app and dataset; never uses the user's port 8080.
// Optional QA dependency: npm install --no-save playwright
const assert = require('node:assert/strict');
const fs = require('node:fs/promises');
const os = require('node:os');
const path = require('node:path');
const net = require('node:net');
const { spawn, spawnSync } = require('node:child_process');
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const project = path.resolve(__dirname, '..');
const delay = milliseconds => new Promise(resolve => setTimeout(resolve, milliseconds));

async function freePort() {
  const probe = net.createServer();
  await new Promise((resolve, reject) => { probe.once('error', reject); probe.listen(0, '127.0.0.1', resolve); });
  const port = probe.address().port;
  await new Promise(resolve => probe.close(resolve));
  return port;
}

(async () => {
  const temporary = await fs.mkdtemp(path.join(os.tmpdir(), 'lognormal-ui-regression-'));
  let server;
  let browser;
  try {
    const executable = path.join(temporary, process.platform === 'win32' ? 'lab.exe' : 'lab');
    const build = spawnSync('go', ['build', '-o', executable, './cmd/app'], { cwd: project, encoding: 'utf8' });
    assert.equal(build.status, 0, build.stderr || 'go build failed');
    const port = await freePort();
    const origin = `http://127.0.0.1:${port}`;
    server = spawn(executable, ['-addr', `127.0.0.1:${port}`, '-data', path.join(temporary, 'data')], { stdio: 'ignore' });
    let ready = false;
    for (let attempt = 0; attempt < 100; attempt++) {
      try { const response = await fetch(origin); if (response.ok) { ready = true; break; } } catch (_) {}
      if (server.exitCode !== null) throw new Error('Temporary app exited before startup');
      await delay(100);
    }
    assert.ok(ready, 'Temporary app did not start');
    browser = await chromium.launch({ channel: process.env.CHROME_CHANNEL || 'chrome', headless: true });
    const page = await browser.newPage({ viewport: { width: 1440, height: 1050 }, reducedMotion: 'reduce' });
    page.setDefaultTimeout(10000);
    const pageErrors = [];
    page.on('pageerror', error => pageErrors.push(error.message));
    const count = () => page.locator('.summary-strip strong').first();

    await page.goto(origin);
    for (const selector of ['#ctl', '#gen', '#files [hx-post]']) {
      assert.ok(await page.locator(selector).evaluateAll(elements => elements.every(element => element.closest('[hx-sync]')?.getAttribute('hx-sync') === '#view:queue all')));
    }
    // Delay a response already calculated for 200 objects, then ask for 120.
    // All updates of #view must share an ordered request queue.
    let release;
    const released = new Promise(resolve => { release = resolve; });
    let responseReady;
    const intercepted = new Promise(resolve => { responseReady = resolve; });
    await page.route('**/view?**', async route => {
      const response = await route.fetch();
      responseReady();
      await released;
      await route.fulfill({ response });
    });
    let generationRequests = 0;
    page.on('request', request => { if (request.method() === 'POST' && request.url() === `${origin}/generate`) generationRequests++; });
    await page.locator('#alpha-number').fill('0.5');
    await intercepted;
    await page.locator('#gen [name=n]').fill('120');
    await page.locator('#gen button').click();
    try {
      await delay(150);
      assert.equal(generationRequests, 0, 'Generation overtook the older view request');
    } finally { release(); }
    await page.waitForFunction(() => document.querySelector('.summary-strip strong')?.textContent === '120');
    assert.equal(await count().innerText(), '120');
    await page.unroute('**/view?**');
    console.log('PASS: shared result queue prevents stale response overwrites');

    // Reverse order: a mutation is already running when the user changes alpha.
    // Cancelling its XHR cannot undo server state, so the view must wait as well.
    let releaseGeneration;
    const generationReleased = new Promise(resolve => { releaseGeneration = resolve; });
    let generationReady;
    const generationIntercepted = new Promise(resolve => { generationReady = resolve; });
    await page.route('**/generate', async route => {
      const response = await route.fetch();
      generationReady();
      await generationReleased;
      await route.fulfill({ response });
    });
    let viewRequests = 0;
    const countViewRequests = request => { if (request.url().startsWith(`${origin}/view?`)) viewRequests++; };
    page.on('request', countViewRequests);
    await page.locator('#gen [name=n]').fill('140');
    await page.locator('#gen button').click();
    await generationIntercepted;
    await page.locator('#alpha-number').fill('0.7');
    try {
      await delay(450);
      assert.equal(viewRequests, 0, 'View overtook a pending mutation');
    } finally { releaseGeneration(); }
    await page.waitForFunction(() => document.querySelector('.summary-strip strong')?.textContent === '140'
      && document.querySelector('.summary-strip > div:nth-child(3) strong')?.textContent === '0.70');
    await page.unroute('**/generate');
    page.off('request', countViewRequests);
    console.log('PASS: view waits for generation and uses the final dataset');

    await page.goto(origin);
    await page.locator('[name=groups]').fill('0');
    await page.locator('.parameter-details').evaluate(element => { element.open = true; });
    await page.locator('[name=noise]').fill('-1');
    await page.locator('#generation-mode').selectOption('random');
    assert.ok(await page.locator('[name=groups]').isDisabled());
    assert.ok(await page.locator('[name=noise]').isDisabled());
    await page.locator('#gen [name=n]').fill('130');
    await page.locator('#gen button').click();
    await page.waitForFunction(() => document.querySelector('.summary-strip strong')?.textContent === '130');
    assert.equal(await page.locator('.error').count(), 0);
    await page.locator('#generation-mode').selectOption('clouds');
    assert.equal(await page.locator('[name=groups]').isDisabled(), false);
    assert.equal(await page.locator('[name=noise]').isDisabled(), false);
    console.log('PASS: hidden group parameters cannot block random generation and re-enable in clouds');

    await page.goto(`${origin}/ops`);
    await page.route('**/ops/calc?**', route => route.abort('connectionfailed'));
    await page.locator('[name=mu1]').fill('1');
    await page.waitForFunction(() => document.querySelector('[data-request-error]')?.textContent.includes('Нет связи'));
    assert.equal(await page.locator('#opsview').getAttribute('aria-busy'), 'false');
    const secondFailure = page.waitForEvent('requestfailed', { predicate: request => request.url().startsWith(`${origin}/ops/calc?`) });
    await page.locator('[name=mu1]').fill('1.1');
    await secondFailure;
    await page.waitForFunction(() => document.querySelector('#opsview').getAttribute('aria-busy') === 'false' && !!document.querySelector('[data-request-error]'));
    assert.equal(await page.locator('[data-request-error]').count(), 1);
    await page.unroute('**/ops/calc?**');
    await page.locator('[name=mu1]').fill('0.2');
    await page.waitForFunction(() => !document.querySelector('[data-request-error]') && document.querySelector('#opsview').getAttribute('aria-busy') === 'false');
    console.log('PASS: network error is visible, deduplicated, clears busy state, and recovers');

    await page.evaluate(() => { window.htmx.config.timeout = 100; });
    await page.route('**/ops/calc?**', async route => { await delay(500); await route.abort().catch(() => {}); });
    await page.locator('[name=mu1]').fill('0.4');
    await page.waitForFunction(() => document.querySelector('[data-request-error]')?.textContent.includes('вовремя'));
    assert.equal(await page.locator('[data-request-error]').count(), 1);
    assert.equal(await page.locator('#opsview').getAttribute('aria-busy'), 'false');
    assert.deepEqual(pageErrors, []);
    console.log('PASS: timeout has an explicit message and clears busy state');
  } finally {
    await browser?.close();
    if (server && server.exitCode === null) {
      const stopped = new Promise(resolve => server.once('exit', resolve));
      server.kill();
      await stopped;
    }
    await fs.rm(temporary, { recursive: true, force: true });
  }
})().catch(error => { console.error(error); process.exitCode = 1; });
