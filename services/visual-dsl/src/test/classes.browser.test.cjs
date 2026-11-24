// Optional real-browser acceptance: point VISUAL_DSL_TEST_URL at this source's Vite server.
// Set VISUAL_DSL_PLAYWRIGHT to an external Playwright installation if needed; no package edits required.
const { test, before, after } = require('node:test');
const assert = require('node:assert/strict');
const enabled = Boolean(process.env.VISUAL_DSL_TEST_URL);
let browser;
before(async () => {
  if (!enabled) return;
  const { chromium } = require(process.env.VISUAL_DSL_PLAYWRIGHT || 'playwright');
  browser = await chromium.launch({ headless: true,
    ...(process.env.VISUAL_DSL_CHROME ? { executablePath: process.env.VISUAL_DSL_CHROME } : {}) });
});
after(async () => { await browser?.close(); });
const labels = ['Class', 'Field', 'Method', 'New', 'Interface', 'Requires', 'Signature', 'WithSignature', 'Satisfies', 'AssertSatisfies'];
const draftKey = 'chariot.visual-dsl.draft.v1';
async function setup() {
  const context = await browser.newContext({ viewport: { width: 1800, height: 1100 } });
  await context.route('**/api/**', route => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ result: 'OK', data: [] }) }));
  const page = await context.newPage();
  page.on('dialog', dialog => dialog.accept());
  await page.goto(process.env.VISUAL_DSL_TEST_URL);
  await page.getByRole('button', { name: /^CLASSES 10/ }).click();
  return { context, page };
}
const palette = (page, label) => page.locator('button').filter({ has: page.locator('span.flex-1').getByText(label, { exact: true }) });
const visual = (page, label) => page.locator('.react-flow__node-logicon').filter({ has: page.getByText(label, { exact: true }) }).last();
async function properties(page, label) {
  await page.getByRole('button', { name: 'fit view', exact: true }).click();
  await visual(page, label).click({ button: 'right' });
  await page.getByRole('button', { name: /Properties\.\.\./ }).click();
  const dialog = page.getByRole('dialog', { name: `${label} Properties`, exact: true });
  await dialog.waitFor();
  return dialog;
}
async function state(page) { return page.evaluate(key => JSON.parse(localStorage.getItem(key)), draftKey); }
async function add(page, label, modifiers) { await palette(page, label).click({ modifiers: modifiers ?? [] }); }

for (const label of labels) test(`Chrome ${label}: palette, actual controls, Save, Cancel, reopen, Escape and focus trap`, { skip: !enabled }, async () => {
  const { page, context } = await setup();
  try {
    if (label === 'Field' || label === 'Requires') {
      const owner = label === 'Field' ? 'Class' : 'Interface';
      await add(page, owner);
      await visual(page, owner).click();
      await page.getByRole('button', { name: /Create Nesting/ }).click();
    }
    await add(page, label, label === 'New' ? ['Shift'] : undefined);
    if (label === 'Field' || label === 'Requires') await page.getByRole('button', { name: /Exit Nesting/ }).click();
    let dialog = await properties(page, label);
    assert.equal(await dialog.locator(':focus').count(), 1);
    const before = (await state(page)).diagram.nodes.find(node => node.data.label === label).data.properties;
    const editable = dialog.locator('input:not([type=checkbox]), textarea').first();
    if (await editable.count()) {
      await editable.fill('changed');
      await dialog.getByRole('button', { name: 'Cancel', exact: true }).click();
      assert.deepEqual((await state(page)).diagram.nodes.find(node => node.data.label === label).data.properties, before);
      dialog = await properties(page, label);
      assert.notEqual(await dialog.locator('input:not([type=checkbox]), textarea').first().inputValue(), 'changed');
    }
    if (label === 'Class' || label === 'Interface') {
      await dialog.getByLabel('Declaration identifier').fill('bad identifier');
      await dialog.getByRole('button', { name: 'Save Properties' }).click();
      assert.match(await dialog.textContent(), /unquoted declaration identifier/);
      assert.equal(await dialog.getByLabel('Declaration identifier').inputValue(), 'bad identifier');
      await dialog.getByLabel('Declaration identifier').fill(label === 'Class' ? 'Account' : 'AccountInterface');
    } else if (label === 'Field') {
      await dialog.getByLabel('Default present', { exact: true }).check();
      await dialog.getByLabel('Default expression', { exact: true }).fill('""');
      await dialog.getByLabel('Default present', { exact: true }).uncheck();
      await dialog.getByLabel('Default present', { exact: true }).check();
      assert.equal(await dialog.getByLabel('Default expression', { exact: true }).inputValue(), '""');
    } else if (label === 'Method') {
      await dialog.getByLabel('Parameter 1 name', { exact: true }).fill('notSelf');
      await dialog.getByRole('button', { name: 'Save Properties' }).click();
      assert.match(await dialog.textContent(), /First method parameter must be self/);
      await dialog.getByLabel('Parameter 1 name', { exact: true }).fill('self');
      await dialog.getByRole('button', { name: 'Add Parameter', exact: true }).click();
      await dialog.getByLabel('Parameter 2 name', { exact: true }).fill('amount');
      await dialog.getByLabel('Parameter 2 value', { exact: true }).fill('0');
      await dialog.getByLabel('Body', { exact: true }).fill('setq(self.value, amount)\nreturn(self.value)');
      await dialog.getByLabel('Function mode').selectOption('expression');
      await dialog.getByLabel('Function expression', { exact: true }).fill('func(self) { return(self.value) }');
      await dialog.getByLabel('Function mode').selectOption('editor');
      assert.match(await dialog.getByLabel('Body', { exact: true }).inputValue(), /setq/);
      await dialog.getByLabel('Include method signature').check();
    } else if (label === 'Signature') {
      await dialog.getByRole('button', { name: 'Add Contract', exact: true }).click();
      await dialog.getByLabel('Parameter contract 1', { exact: true }).selectOption('N');
      await dialog.getByRole('button', { name: 'Add Contract', exact: true }).click();
      await dialog.getByLabel('Parameter contract 2', { exact: true }).selectOption('S');
      await dialog.getByRole('button', { name: 'Remove contract 1', exact: true }).click();
    } else if (label === 'New') {
      await dialog.getByLabel('Attributes present', { exact: true }).check();
      await dialog.getByLabel('Attributes expression', { exact: true }).fill('mapValue("value", 0)');
    }
    await dialog.getByRole('button', { name: 'Save Properties' }).click();
    await dialog.waitFor({ state: 'hidden' });
    const saved = (await state(page)).diagram.nodes.find(node => node.data.label === label).data.properties;
    dialog = await properties(page, label);
    if (label === 'Method') assert.equal(await dialog.getByLabel('Parameter 2 value', { exact: true }).inputValue(), '0');
    if (label === 'Field') assert.equal(await dialog.getByLabel('Default expression', { exact: true }).inputValue(), '""');
    if (label === 'Signature') assert.equal(await dialog.getByLabel('Parameter contract 1', { exact: true }).inputValue(), 'S');
    await dialog.getByRole('button', { name: 'Delete', exact: true }).focus();
    await page.keyboard.press('Tab');
    assert.equal(await dialog.locator(':focus').count(), 1);
    await page.keyboard.press('Shift+Tab');
    assert.equal(await dialog.getByRole('button', { name: 'Delete', exact: true }).evaluate(element => element === document.activeElement), true);
    await page.keyboard.press('Escape');
    await dialog.waitFor({ state: 'hidden' });
    assert.equal(await visual(page, label).evaluate(element => element === document.activeElement), true);
    assert.deepEqual((await state(page)).diagram.nodes.find(node => node.data.label === label).data.properties, saved);
    await page.reload();
    assert.deepEqual((await state(page)).diagram.nodes.find(node => node.data.label === label).data.properties, saved);
    dialog = await properties(page, label);
    await dialog.getByRole('button', { name: 'Delete', exact: true }).click();
    assert.equal((await state(page)).diagram.nodes.some(node => node.data.label === label), false);
  } finally { await context.close(); }
});

test('Chrome connector nesting repairs imported root members and rejects wrong declarations without mutation', { skip: !enabled }, async () => {
  const { page, context } = await setup();
  const messages = [];
  page.on('dialog', dialog => messages.push(dialog.message()));
  try {
    // Imported incomplete graphs must remain editable, never silently normalized.
    const imported = await state(page);
    const make = (label, properties, x) => ({ id: label.toLowerCase(), type: 'logicon', position: { x, y: 200 }, data: { label, icon: '', category: 'classes', properties } });
    imported.diagram.nodes.push(make('Class', { identifier: 'Example' }, 100), make('Interface', { identifier: 'ExampleInterface' }, 400),
      make('Field', { memberName: 'value', contract: 'V', hasDefault: false, defaultExpression: 'null' }, 700));
    await page.evaluate(({ key, value }) => localStorage.setItem(key, JSON.stringify(value)), { key: draftKey, value: imported });
    await page.reload();
    await page.getByRole('button', { name: 'fit view', exact: true }).click();
    assert.match(await visual(page, 'Field').textContent(), /direct child of Class/);
    const source = visual(page, 'Interface').locator('.react-flow__handle[data-handleid="right"]');
    const target = visual(page, 'Field').locator('.react-flow__handle[data-handleid="left"]');
    await source.dragTo(target);
    assert.equal((await state(page)).diagram.nestingRelations.length, 0);
    assert.equal((await state(page)).diagram.edges.length, 0);
    assert.match(messages.pop(), /direct child of Class|only Requires/);
    await visual(page, 'Class').locator('.react-flow__handle[data-handleid="right"]').dragTo(target);
    assert.deepEqual((await state(page)).diagram.nestingRelations, [{ parentId: 'class', childId: 'field', order: 0 }]);
    assert.equal((await state(page)).diagram.edges.length, 1);
    assert.ok(!(await visual(page, 'Field').textContent()).includes('direct child'));
  } finally { await context.close(); }
});

test('Chrome palette -> declarations/nested members -> methods -> generate -> named Save/reload', { skip: !enabled }, async () => {
  const { page, context } = await setup();
  const messages = [];
  page.on('dialog', dialog => messages.push(dialog.message()));
  try {
    await add(page, 'Field');
    assert.equal((await state(page)).diagram.nodes.some(node => node.data.label === 'Field'), false);
    assert.match(messages.pop(), /Select a Class/);
    await add(page, 'Class');
    await visual(page, 'Class').click();
    await page.getByRole('button', { name: /Create Nesting/ }).click();
    await add(page, 'Method');
    assert.equal((await state(page)).diagram.nodes.some(node => node.data.label === 'Method'), false);
    assert.match(messages.pop(), /only Field/);
    await add(page, 'Field');
    await page.getByRole('button', { name: /Exit Nesting/ }).click();
    await add(page, 'Method');
    await add(page, 'Interface');
    await page.getByRole('button', { name: 'fit view', exact: true }).click();
    await visual(page, 'Interface').click();
    await page.getByRole('button', { name: /Create Nesting/ }).click();
    await add(page, 'Requires', ['Shift']);
    await page.getByRole('button', { name: /Exit Nesting/ }).click();
    await page.getByRole('button', { name: /Generate .ch Code/ }).click();
    await page.getByRole('button', { name: '🚀 Generate Code', exact: true }).click();
    const code = await page.locator('pre').textContent();
    assert.match(code, /class\(Example\)\s*\{/);
    assert.match(code, /field\("value", "V"\)/);
    assert.match(code, /method\(Example, "getValue", func\(self\)/);
    assert.match(code, /interface\(ExampleInterface\)/);
    assert.match(code, /requires\("getValue", signature\(array\(\), "V"\)\)/);
    await page.getByRole('button', { name: /^Close$/ }).click();
    await page.getByRole('button', { name: '💾 Save', exact: true }).click();
    const saved = await state(page);
    assert.equal(saved.diagram.nestingRelations.length, 2);
    await page.reload();
    const restored = await state(page);
    assert.deepEqual(restored.diagram.nodes, saved.diagram.nodes);
    assert.deepEqual(restored.diagram.nestingRelations, saved.diagram.nestingRelations);
    console.log('Generated class/interface program through actual palette and nesting UI:\n' + code.split('// @')[0].slice(0, 1000));
  } finally { await context.close(); }
});