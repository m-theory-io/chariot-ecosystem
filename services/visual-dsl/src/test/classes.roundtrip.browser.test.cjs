// Opt-in: VISUAL_DSL_ROUNDTRIP=1 VISUAL_DSL_TEST_URL=http://localhost:5186
// Uses VISUAL_DSL_PLAYWRIGHT / VISUAL_DSL_CHROME like classes.browser.test.cjs.
// Real: source frontend, unchanged landing iframe, current Charioteer HTTP proxy,
// clipboard, Monaco and freshly built Go CLI. Stubbed: login/session/files/runtime
// metadata, and the HTTP execution envelope (NOT interpreter execution).
const { test, before, after } = require('node:test');
const assert = require('node:assert/strict');
const { execFile, execFileSync, spawn } = require('node:child_process');
const { once } = require('node:events');
const { mkdtempSync, readFileSync, writeFileSync, rmSync, mkdirSync, cpSync } = require('node:fs');
const http = require('node:http');
const { tmpdir } = require('node:os');
const { resolve, join } = require('node:path');
const { promisify } = require('node:util');
const run = promisify(execFile);
const enabled = process.env.VISUAL_DSL_ROUNDTRIP === '1' && !!process.env.VISUAL_DSL_TEST_URL;
const root = resolve(__dirname, '../../../..');
const fixture = join(root, 'packages/chariot-codegen/test/fixtures/classes-complete.json');
const draftKey = 'chariot.visual-dsl.draft.v1';
let browser, temp, backend, shell, charioteer, origin, cli;
const executions = [];
const metadataRequests = new Set();
const listen = async server => {
  server.listen(0, '127.0.0.1'); await once(server, 'listening');
  return `http://127.0.0.1:${server.address().port}`;
};
const close = server => server && new Promise(resolve => { server.close(resolve); server.closeAllConnections(); });
function proxy(req, res, target, path = req.url) {
  const upstream = http.request(new URL(path, target), { method: req.method,
    headers: { ...req.headers, host: new URL(target).host } }, response => {
    res.writeHead(response.statusCode, response.headers); response.pipe(res);
  });
  upstream.on('error', error => { res.writeHead(502); res.end(error.message); });
  req.pipe(upstream);
}
before(async () => {
  if (!enabled) return;
  temp = mkdtempSync(join(tmpdir(), 'classes-roundtrip-'));
  cli = join(temp, 'chariotctl');
  execFileSync('go', ['build', '-o', cli, './services/go-chariot/cmd/chariotctl'], { cwd: root });
  execFileSync('go', ['build', '-o', join(temp, 'charioteer'), './services/charioteer'], { cwd: root });
  // Isolate all Charioteer startup files from the user's working tree.
  cpSync(join(root, 'services/charioteer/templates'), join(temp, 'templates'), { recursive: true });
  mkdirSync(join(temp, 'files'));
  backend = http.createServer(async (req, res) => {
    res.setHeader('Content-Type', 'application/json');
    try {
      if (req.url.split('?')[0] === '/api/execute') {
        const chunks = []; for await (const chunk of req) chunks.push(chunk);
        const { program } = JSON.parse(Buffer.concat(chunks).toString());
        const path = join(temp, `execution-${executions.length}.ch`);
        writeFileSync(path, program);
        const { stdout, stderr } = await run(cli, ['-f', path], { cwd: temp, timeout: 30000 });
        const record = { program, stdout: stdout.trim(), stderr, realGo: true };
        executions.push(record);
        // CLI values use JSON-compatible output for these acceptance examples.
        res.end(JSON.stringify({ result: 'OK', data: JSON.parse(record.stdout) }));
      } else {
        metadataRequests.add(req.url.split('?')[0]);
        let data = [];
        if (req.url.includes('/session/profile')) data = { sandbox_enabled: false, sandbox_scopes: ['global'], sandbox_scope_default: 'global' };
        if (req.url.includes('/runtime/inspect')) data = {};
        res.end(JSON.stringify({ result: 'OK', data }));
      }
    } catch (error) {
      res.statusCode = 500;
      res.end(JSON.stringify({ result: 'ERROR', data: error.stderr || error.message }));
    }
  });
  const backendURL = await listen(backend);
  // Charioteer has no port-zero discovery; reserve a free port, release, then
  // wait for its actual startup message (not a fixed startup sleep).
  const reservation = http.createServer();
  const reservedURL = await listen(reservation);
  await close(reservation);
  charioteer = spawn(join(temp, 'charioteer'), ['-port', new URL(reservedURL).port,
    '-backend', backendURL, '-mcp-data', join(temp, 'metadata')], { cwd: temp });
  await new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(Error('Charioteer startup timeout')), 15000);
    charioteer.once('error', error => { clearTimeout(timer); reject(error); });
    charioteer.once('exit', code => { clearTimeout(timer); reject(Error(`Charioteer exited: ${code}`)); });
    charioteer.stderr.on('data', chunk => {
      if (/Starting.*server|Server.*starting|Listening|listening/.test(chunk.toString())) { clearTimeout(timer); resolve(); }
    });
  });
  shell = http.createServer((req, res) => {
    if (req.url === '/') {
      res.setHeader('Content-Type', 'text/html');
      res.end(readFileSync(join(root, 'infrastructure/docker/nginx/landing.html')));
    } else if (req.url.startsWith('/charioteer/') || /^\/(editor-state|chariot-codegen)\.js/.test(req.url)) {
      proxy(req, res, reservedURL);
    } else if (req.url.startsWith('/api/')) {
      proxy(req, res, backendURL);
    } else {
      proxy(req, res, process.env.VISUAL_DSL_TEST_URL, req.url.replace(/^\/visual-dsl\//, '/'));
    }
  });
  origin = await listen(shell);
  const { chromium } = require(process.env.VISUAL_DSL_PLAYWRIGHT || 'playwright');
  browser = await chromium.launch({ headless: true,
    ...(process.env.VISUAL_DSL_CHROME ? { executablePath: process.env.VISUAL_DSL_CHROME } : {}) });
});
after(async () => {
  await browser?.close();
  await close(shell); await close(backend);
  if (charioteer && charioteer.exitCode === null) {
    const exited = once(charioteer, 'exit'); charioteer.kill(); await exited;
  }
  if (temp) rmSync(temp, { recursive: true, force: true });
});
async function setup() {
  const context = await browser.newContext({ permissions: ['clipboard-read', 'clipboard-write'], viewport: { width: 1800, height: 1100 } });
  await context.addInitScript(() => {
    localStorage.setItem('chariot_token', 'ACCEPTANCE-NOT-A-SECRET');
    localStorage.setItem('chariot_user', 'classes-acceptance');
  });
  const page = await context.newPage();
  page.on('dialog', dialog => dialog.accept());
  await page.goto(origin);
  return { context, page, frame: await app(page, '/visual-dsl/') };
}
async function app(page, path) {
  await page.waitForFunction(path => document.querySelector('#microframe').contentWindow.location.pathname === path, path);
  const frame = page.frames().find(frame => new URL(frame.url()).pathname === path);
  if (path === '/visual-dsl/') await frame.getByRole('button', { name: /^CLASSES 10/ }).waitFor();
  else await frame.waitForFunction(() => typeof editor === 'object' && workspaceInitialized);
  return frame;
}
const state = frame => frame.evaluate(key => JSON.parse(localStorage.getItem(key)), draftKey);
const visual = (frame, label, id) => id ? frame.locator(`.react-flow__node[data-id="${id}"]`)
  : frame.locator('.react-flow__node-logicon').filter({ has: frame.getByText(label, { exact: true }) }).last();
async function properties(frame, label, id) {
  await frame.getByRole('button', { name: 'fit view', exact: true }).click();
  await visual(frame, label, id).click({ button: 'right' });
  await frame.getByRole('button', { name: /Properties\.\.\./ }).click();
  const dialog = frame.getByRole('dialog', { name: `${label} Properties`, exact: true });
  await dialog.waitFor(); return dialog;
}
async function save(dialog) { await dialog.getByRole('button', { name: 'Save Properties' }).click(); await dialog.waitFor({ state: 'hidden' }); }
async function add(frame, label) {
  await frame.locator('button').filter({ has: frame.locator('span.flex-1').getByText(label, { exact: true }) }).click();
}
async function generate(frame) {
  await frame.getByRole('button', { name: /Generate .ch Code/ }).click();
  await frame.getByRole('button', { name: '🚀 Generate Code', exact: true }).click();
  const code = await frame.locator('pre').textContent();
  await frame.getByRole('button', { name: '📋 Copy', exact: true }).click();
  assert.equal(await frame.evaluate(() => navigator.clipboard.readText()), code);
  await frame.getByRole('button', { name: 'Close', exact: true }).click();
  return code;
}
async function executeInEditor(page, code, expected) {
  await page.getByRole('button', { name: '⚙️ Charioteer', exact: true }).click();
  const frame = await app(page, '/charioteer/editor');
  // Paste through Monaco's actual clipboard action, not editor.setValue(code).
  await frame.evaluate(async () => {
    editor.setValue(''); editor.focus();
    await editor.getAction('editor.action.clipboardPasteAction').run();
  });
  assert.equal(await frame.evaluate(() => editor.getValue()), code);
  await frame.locator('#streamingToggle').uncheck();
  const response = page.waitForResponse(response => response.url().endsWith('/charioteer/api/execute') && response.request().method() === 'POST');
  await frame.locator('#runButton').click();
  const executed = await response;
  assert.equal(executed.status(), 200, await executed.text());
  assert.deepEqual(await executed.json(), { result: 'OK', data: expected });
  await frame.waitForFunction(() => document.getElementById('outputContent').textContent.includes('Result:'));
  assert.deepEqual(JSON.parse((await frame.locator('#outputContent').textContent()).split('Result: ')[1]), expected);
  // Charioteer may issue a background listFunctions() execution immediately
  // after Run; correlate proof with the submitted source, not completion order.
  const execution = executions.findLast(record => record.program === code);
  assert.ok(execution, 'The exact copied program must reach the real CLI');
  assert.equal(execution.realGo, true); assert.equal(execution.stderr, '');
  assert.deepEqual(JSON.parse(execution.stdout), expected);
  console.log(`Real Go CLI via Charioteer Run: ${execution.stdout} (${code.length} generated characters)`);
}
async function switchBack(page, saved) {
  await page.getByRole('button', { name: '📊 Visual DSL', exact: true }).click();
  const frame = await app(page, '/visual-dsl/');
  const restored = await state(frame);
  assert.deepEqual(restored.diagram.nodes, saved.diagram.nodes);
  assert.deepEqual(restored.diagram.edges, saved.diagram.edges);
  assert.deepEqual(restored.diagram.nestingRelations, saved.diagram.nestingRelations);
  assert.equal(restored.diagram.name, saved.diagram.name);
  return frame;
}
test('Chrome palette author -> Copy -> Charioteer Run (real Go) -> iframe draft restore -> dialog edit', { skip: !enabled }, async () => {
  const { context, page, frame } = await setup();
  try {
    await frame.getByRole('button', { name: /^CLASSES 10/ }).click();
    await add(frame, 'Class');
    await visual(frame, 'Class').click();
    await frame.getByRole('button', { name: /Create Nesting/ }).click();
    await add(frame, 'Field');
    await frame.getByRole('button', { name: /Exit Nesting/ }).click();
    let dialog = await properties(frame, 'Field');
    await dialog.getByLabel('Default present', { exact: true }).check();
    await dialog.getByLabel('Default expression', { exact: true }).fill('42'); await save(dialog);
    await add(frame, 'Method');
    dialog = await properties(frame, 'Method');
    await dialog.getByLabel('Body', { exact: true }).fill('return(self.value)');
    await dialog.getByLabel('Include method signature').check();
    await dialog.getByLabel('Signature expression', { exact: true }).fill('signature(array("V"), "V")'); await save(dialog);
    await add(frame, 'Interface');
    await frame.getByRole('button', { name: 'fit view', exact: true }).click();
    await visual(frame, 'Interface').click();
    await frame.getByRole('button', { name: /Create Nesting/ }).click();
    await add(frame, 'Requires');
    await frame.getByRole('button', { name: /Exit Nesting/ }).click();
    await add(frame, 'Satisfies');
    dialog = await properties(frame, 'Satisfies');
    await dialog.getByLabel('Value expression', { exact: true }).fill('new(Example, map("value", new(Example).getValue()))');
    await dialog.getByLabel('Interface expression', { exact: true }).fill('ExampleInterface'); await save(dialog);
    const code = await generate(frame);
    assert.match(code, /field\("value", "V", 42\)/);
    const saved = await state(frame); // Deliberately never click named Save.
    await executeInEditor(page, code, true);
    const restored = await switchBack(page, saved);
    dialog = await properties(restored, 'Field');
    assert.equal(await dialog.getByLabel('Default expression', { exact: true }).inputValue(), '42');
    await dialog.getByLabel('Default expression', { exact: true }).fill('43'); await save(dialog);
    assert.match(await generate(restored), /field\("value", "V", 43\)/);
    // Charioteer also survives iframe destruction, including the pasted program.
    await page.getByRole('button', { name: '⚙️ Charioteer', exact: true }).click();
    const editorFrame = await app(page, '/charioteer/editor');
    assert.equal(await editorFrame.evaluate(() => editor.getValue()), code);
  } finally { await context.close(); }
});
test('Chrome shared all-ten fixture -> dialog edit -> Copy -> real Go result -> editable restore and refresh', { skip: !enabled }, async () => {
  const { context, page, frame } = await setup();
  try {
    // Load the checked-in shared fixture through the public import UI, unedited.
    await frame.getByRole('button', { name: /Paste JSON/ }).click();
    await frame.getByPlaceholder('Paste your diagram JSON here...').fill(readFileSync(fixture, 'utf8'));
    await frame.getByRole('button', { name: 'Load Diagram', exact: true }).click();
    let dialog = await properties(frame, 'Method', 'value-method');
    await dialog.getByLabel('Body', { exact: true }).fill('return(add(self.count, 0))'); await save(dialog);
    const code = await generate(frame);
    for (const name of ['class', 'field', 'method', 'new', 'interface', 'requires', 'signature', 'withSignature', 'satisfies', 'assertSatisfies']) {
      assert.ok(code.includes(`${name}(`), name);
    }
    const saved = await state(frame);
    await executeInEditor(page, code, [42, 42, true, false, 7, 0]);
    let restored = await switchBack(page, saved);
    dialog = await properties(restored, 'Method', 'value-method');
    assert.equal(await dialog.getByLabel('Body', { exact: true }).inputValue(), 'return(add(self.count, 0))');
    await dialog.getByLabel('Body', { exact: true }).fill('return(add(self.count, 1))'); await save(dialog);
    const edited = await state(restored);
    await page.reload(); restored = await app(page, '/visual-dsl/');
    assert.deepEqual((await state(restored)).diagram.nodes, edited.diagram.nodes);
    assert.deepEqual((await state(restored)).diagram.nestingRelations, edited.diagram.nestingRelations);
    const editedCode = await generate(restored);
    assert.match(editedCode, /return\(add\(self.count, 1\)\)/);
    await executeInEditor(page, editedCode, [42, 43, true, false, 7, 0]);
    console.log('Stubbed metadata endpoints: ' + [...metadataRequests].sort().join(', '));
  } finally { await context.close(); }
});