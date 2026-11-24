// Optional real-browser suite. Requires a running Charioteer and Playwright (see docs/EditorDrafts.md).
const { test, before, after } = require('node:test');
const assert = require('node:assert/strict');
const { chromium } = require(process.env.CHARIOTEER_PLAYWRIGHT || 'playwright');
const base = process.env.CHARIOTEER_TEST_URL || 'http://localhost:18080';
let browser;
before(async () => {
    browser = await chromium.launch({ headless: true,
        ...(process.env.CHARIOTEER_CHROME ? { executablePath: process.env.CHARIOTEER_CHROME } : {}) });
});
after(async () => { await browser?.close(); });
async function context() {
    const ctx = await browser.newContext({ permissions: ['clipboard-read', 'clipboard-write'], viewport: { width: 1800, height: 1000 } });
    await ctx.route('**/api/**', route => {
        const path = new URL(route.request().url()).pathname;
        let data = [];
        if (path.endsWith('/session/profile')) data = { sandbox_scopes: ['global', 'sandbox', 'listeners', 'bootstrap'], sandbox_scope_default: 'global' };
        if (path.endsWith('/runtime/inspect')) data = {};
        return route.fulfill({ contentType: 'application/json', body: JSON.stringify({ result: 'OK', data }) });
    });
    return ctx;
}
async function ready(page) {
    await page.waitForFunction(() => typeof editor === 'object' && workspaceInitialized);
}
test('Chrome: native paste once; menu paste once even if host execCommand would dispatch paste and return false; denied menu falls back to keyboard', async () => {
    const ctx = await context();
    try {
        const page = await ctx.newPage();
        await page.goto(base + '/charioteer/editor'); await ready(page);
        await page.evaluate(async () => {
            editor.focus(); await navigator.clipboard.writeText('declare(x, "N", 42)');
            window.nativePastes = 0;
            document.addEventListener('paste', () => window.nativePastes++, true);
        });
        await page.keyboard.press(process.platform === 'darwin' ? 'Meta+v' : 'Control+v');
        await page.waitForFunction(() => editor.getValue().length > 0);
        assert.deepEqual(await page.evaluate(() => [editor.getValue(), window.nativePastes]), ['declare(x, "N", 42)', 1]);
        await page.evaluate(async () => {
            editor.setValue(''); editor.focus();
            window.nativeAttempts = 0;
            const original = document.execCommand.bind(document);
            document.execCommand = (cmd, ...args) => {
                if (cmd !== 'paste') return original(cmd, ...args);
                window.nativeAttempts++;
                const data = new DataTransfer(); data.setData('text/plain', 'MENU');
                editor.getDomNode().querySelector('textarea').dispatchEvent(new ClipboardEvent('paste', { clipboardData: data, bubbles: true, cancelable: true }));
                return false;
            };
            Object.defineProperty(navigator.clipboard, 'readText', { configurable: true,
                value: () => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve('MENU')))) });
            editor.trigger('test', 'editor.action.clipboardPasteAction');
        });
        await page.waitForFunction(() => editor.getValue() === 'MENU');
        assert.equal(await page.evaluate(() => window.nativeAttempts), 0);
        await page.evaluate(async () => {
            editor.setValue('');
            Object.defineProperty(navigator.clipboard, 'readText', { configurable: true, value: async () => { throw Error('permission denied'); } });
            await editor.getAction('editor.action.clipboardPasteAction').run();
        });
        assert.equal(await page.evaluate(() => editor.getValue()), '');
        assert.match(await page.locator('#draftWarning').textContent(), /Ctrl\/Cmd\+V/);
        // Native keyboard paste must NOT depend on readText permission.
        await page.evaluate(async () => { await navigator.clipboard.writeText('KEYBOARD'); editor.focus(); });
        await page.keyboard.press(process.platform === 'darwin' ? 'Meta+v' : 'Control+v');
        await page.waitForFunction(() => editor.getValue() === 'KEYBOARD');
        assert.equal(await page.evaluate(() => window.nativePastes), 2);
    } finally { await ctx.close(); }
});
test('Chrome: iframe replacement and refresh restore all code tabs, dirty deletions, cursor, scroll, active view, scopes, split and layout; New stays new', async () => {
    const ctx = await context();
    try {
        await ctx.addInitScript(() => {
            localStorage.setItem('chariot_token', 'TEST-NOT-A-SECRET');
            localStorage.setItem('chariot_user', 'draft-test-user');
        });
        const page = await ctx.newPage();
        await page.route('**/test-shell', route => route.fulfill({ contentType: 'text/html', body:
            '<iframe id="app" style="width:1200px;height:900px" src="/charioteer/editor"></iframe>' }));
        await page.goto(base + '/test-shell');
        let frame = page.frames().find(f => f.url().endsWith('/charioteer/editor'));
        await ready(frame);
        const expected = await frame.evaluate(() => {
            const a = createTabRecord({ id: 'file', scope: 'sandbox', fileName: 'file.ch',
                content: Array.from({ length: 150 }, (_, i) => '// line ' + i).join('\n'), originalContent: 'saved' });
            const b = createTabRecord({ id: 'fn', scope: FUNCTION_LIBRARY_SCOPE, fileName: 'myFunction', content: '', originalContent: 'deleted code' });
            const d = createTabRecord({ id: 'diagram', scope: 'global', fileName: 'generated.ch', content: '// generated\n42', originalContent: '' });
            d.sourceType = 'diagram'; d.diagramSourceName = 'myDiagram';
            for (const tab of [a, b, d]) registerTab(tab);
            renderPrimaryTabs(); activateFileTab(a.id, { force: true, skipPersist: true });
            editor.setSelection(new monaco.Selection(100, 2, 101, 4)); editor.setScrollTop(1700);
            persistActiveTabContent();
            pinTabInReference(d.id);
            document.getElementById('leftPanel').style.width = '333px';
            document.getElementById('bottomPanel').style.height = '222px';
            document.getElementById('streamingToggle').checked = false;
            flushEditorDraft();
            return { selection: editor.getSelection(), top: editor.getScrollTop(), text: editor.getValue() };
        });
        // Same iframe src replacement used by shared landing; no landing implementation changes.
        await page.locator('#app').evaluate(el => { el.src = 'about:blank'; });
        await page.waitForFunction(() => document.querySelector('iframe').contentWindow.location.href === 'about:blank');
        await page.locator('#app').evaluate(el => { el.src = '/charioteer/editor'; });
        await page.waitForFunction(() => document.querySelector('iframe').contentWindow.location.pathname === '/charioteer/editor');
        frame = page.frames().find(f => f.url().endsWith('/charioteer/editor')); await ready(frame);
        const restored = await frame.evaluate(() => ({ text: editor.getValue(), selection: editor.getSelection(), top: editor.getScrollTop(),
            order: tabOrder, active: activeFileTabId, reference: referenceTabId, view: currentTab, scope: currentFileBrowserScope,
            fn: openFileTabs.get('fn'), diagram: openFileTabs.get('diagram').diagramSourceName,
            width: document.getElementById('leftPanel').style.width, height: document.getElementById('bottomPanel').style.height,
            stream: document.getElementById('streamingToggle').checked }));
        assert.equal(restored.text, expected.text); assert.deepEqual(restored.selection, expected.selection);
        assert.equal(restored.top, expected.top); assert.deepEqual(restored.order, ['file', 'fn']);
        assert.equal(restored.active, 'file'); assert.equal(restored.reference, 'diagram'); assert.equal(restored.scope, 'sandbox');
        assert.equal(restored.fn.content, ''); assert.equal(restored.fn.isDirty, true);
        assert.equal(restored.diagram, 'myDiagram'); assert.equal(restored.width, '333px'); assert.equal(restored.height, '222px'); assert.equal(restored.stream, false);
        await frame.evaluate(() => { activateFileTab('fn'); flushEditorDraft(); });
        await frame.goto(base + '/charioteer/editor'); await ready(frame);
        assert.deepEqual(await frame.evaluate(() => [currentTab, activeFileTabId, editor.getValue(), isFileModified]), ['functions', 'fn', '', true]);
        await frame.evaluate(() => { document.getElementById('filesTab').click(); newFile(); flushEditorDraft(); });
        await frame.goto(base + '/charioteer/editor'); await ready(frame);
        assert.equal(await frame.evaluate(() => editor.getValue()), '// New Chariot Script\n');
        assert.equal(await frame.evaluate(() => openFileTabs.size), 4);
        assert.equal(await frame.evaluate(() => document.getElementById('draftWarning')?.textContent || ''), '');
    } finally { await ctx.close(); }
});

test('Chrome: username changes isolate drafts; logout saves before clearing; login restores without server overwrite', async () => {
    const ctx = await context();
    try {
        await ctx.route('**/login', route => route.fulfill({ contentType: 'application/json',
            body: JSON.stringify({ result: 'OK', data: { token: 'TEST-TOKEN' } }) }));
        const page = await ctx.newPage();
        await page.goto(base + '/charioteer/editor'); await ready(page);
        async function loginAs(user) {
            await page.locator('#usernameInput').fill(user);
            await page.locator('#passwordInput').fill('test-only');
            await page.locator('#loginButton').click();
            await page.waitForFunction(name => currentUser === name && !document.getElementById('loginButton').disabled, user);
        }
        await loginAs('alice');
        await page.evaluate(() => { newFile(); editor.setValue('alice unsaved'); flushEditorDraft(); logout(); });
        assert.equal(await page.evaluate(() => editor.getValue()), '');
        await loginAs('bob');
        assert.equal(await page.evaluate(() => openFileTabs.size), 0);
        await page.evaluate(() => { newFile(); editor.setValue('bob unsaved'); flushEditorDraft(); logout(); });
        await loginAs('alice');
        assert.deepEqual(await page.evaluate(() => [editor.getValue(), openFileTabs.size, isFileModified]), ['alice unsaved', 1, true]);
        const draft = await page.evaluate(() => localStorage.getItem(draftStore.key));
        assert.equal(draft.includes('TEST-TOKEN'), false); assert.equal(draft.includes('test-only'), false);
        assert.equal(draft.includes('bob unsaved'), false);
    } finally { await ctx.close(); }
});

test('Chrome: standalone editor selection restores; quota/corrupt failures warn and keep last stored bytes', async () => {
    const ctx = await context();
    try {
        const page = await ctx.newPage();
        await page.goto(base + '/charioteer/editor'); await ready(page);
        await page.evaluate(() => {
            editor.setValue('standalone unsaved'); editor.setSelection(new monaco.Selection(1, 2, 1, 8)); flushEditorDraft();
        });
        await page.reload(); await ready(page);
        assert.deepEqual(await page.evaluate(() => [editor.getValue(), editor.getSelection().startColumn, editor.getSelection().endColumn]), ['standalone unsaved', 2, 8]);
        const previous = await page.evaluate(() => localStorage.getItem(draftStore.key));
        await page.evaluate(() => {
            const write = Storage.prototype.setItem;
            window.restoreStorage = () => { Storage.prototype.setItem = write; };
            Storage.prototype.setItem = function (key, value) {
                if (key.startsWith('charioteer.editor.draft.')) throw new DOMException('full', 'QuotaExceededError');
                return write.call(this, key, value);
            };
            editor.setValue('new unsaved code'); flushEditorDraft();
        });
        assert.match(await page.locator('#draftWarning').textContent(), /could not be saved/);
        assert.equal(await page.evaluate(() => localStorage.getItem(draftStore.key)), previous);
        await page.evaluate(() => {
            window.restoreStorage(); flushEditorDraft();
            localStorage.setItem(draftStore.key, '{corrupt');
            draftStore = null; // Simulate corruption between documents, without the outgoing pagehide writer repairing it.
        });
        await page.reload(); await ready(page);
        assert.match(await page.locator('#draftWarning').textContent(), /could not be restored/);
        await page.evaluate(() => { editor.setValue('do not overwrite corrupt draft'); flushEditorDraft(); });
        assert.equal(await page.evaluate(() => localStorage.getItem(draftStore.key)), '{corrupt');
    } finally { await ctx.close(); }
});

test('Chrome: replacing Monaco with MCP retains free-form drafts; rebuilding does not duplicate New handlers or persist connection secrets', async () => {
    const ctx = await context();
    try {
        const page = await ctx.newPage();
        await page.goto(base + '/charioteer/editor'); await ready(page);
        await page.evaluate(() => {
            editor.setValue('free-form draft');
            document.getElementById('mcpTab').click();
            document.getElementById('mcpEnvInput').value = 'SECRET=DO-NOT-PERSIST';
            document.getElementById('mcpTokenInput').value = 'DO-NOT-PERSIST';
            flushEditorDraft();
        });
        assert.equal(await page.evaluate(() => localStorage.getItem(draftStore.key).includes('DO-NOT-PERSIST')), false);
        await page.locator('#filesTab').click();
        assert.equal(await page.evaluate(() => editor.getValue()), 'free-form draft');
        await page.locator('#mcpTab').click(); await page.locator('#filesTab').click();
        assert.equal(await page.evaluate(() => editor.getValue()), 'free-form draft');
        const before = await page.evaluate(() => { authToken = 'TEST'; return openFileTabs.size; });
        // Exercise the registered listener directly; existing toolbar CSS can overlap the button.
        await page.locator('#newButton').evaluate(button => button.click());
        assert.equal(await page.evaluate(() => openFileTabs.size), before + 1);
        assert.equal(await page.evaluate(() => monaco.editor.getEditors().length), 2);
        assert.equal(await page.evaluate(() => openFileTabs.values().next().value.content), 'free-form draft');
    } finally { await ctx.close(); }
});
