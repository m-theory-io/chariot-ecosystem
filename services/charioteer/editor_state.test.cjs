const { test } = require('node:test');
const assert = require('node:assert/strict');
const { createStore, project, installPaste } = require('./templates/editor-state.js');

const sample = () => ({ version: 1, tabs: [{ id: 'a', scope: 'sandbox', fileName: 'draft.ch',
    content: '', originalContent: 'saved code', isDirty: false, loaded: true, token: 'NEVER' }],
    order: ['a'], activeId: 'a', view: 'files', fields: { createPlanCode: 'unsaved plan',
        passwordInput: 'NEVER', mcpEnvInput: 'NEVER', mcpPayloadInput: 'NEVER' }, authToken: 'NEVER' });
function storage() {
    const values = new Map();
    return { values, getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value) };
}
test('restore before write; preserve unsaved deletion, scope, ordering, clean baseline and only allowed fields', () => {
    const disk = storage();
    const warnings = [];
    const store = createStore(() => disk, 'alice', message => warnings.push(message));
    assert.equal(store.save(sample()), false);
    assert.equal(store.load(), null);
    assert.equal(store.save(sample()), true);
    const reload = createStore(() => disk, 'alice', assert.fail).load();
    assert.equal(reload.tabs[0].content, '');
    assert.equal(reload.tabs[0].originalContent, 'saved code');
    assert.equal(reload.tabs[0].isDirty, true);
    assert.equal(reload.tabs[0].scope, 'sandbox');
    assert.equal(reload.fields.createPlanCode, 'unsaved plan');
    assert.equal(JSON.stringify(reload).includes('NEVER'), false);
    assert.deepEqual(warnings, []);
    assert.equal(createStore(() => disk, 'bob', assert.fail).load(), null);
    assert.equal(createStore(() => disk, null, assert.fail).load(), null);
});
test('corrupt/unknown snapshots are kept byte-for-byte and cannot be overwritten by autosave', () => {
    for (const raw of ['{broken', '{"version":2,"tabs":[]}', '{"version":1,"tabs":[{}],"order":[]}']) {
        const disk = storage(); const warnings = [];
        const store = createStore(() => disk, 'alice', message => warnings.push(message));
        disk.setItem(store.key, raw);
        assert.equal(store.load(), null);
        assert.equal(store.save(sample()), false);
        assert.equal(disk.getItem(store.key), raw);
        assert.equal(warnings.length, 1);
    }
});
test('quota failure retains previous snapshot, warns once, retries after recovery', () => {
    const disk = storage(); const warnings = [];
    const store = createStore(() => disk, 'alice', message => warnings.push(message));
    store.load(); store.save(sample());
    const old = disk.getItem(store.key); const write = disk.setItem;
    disk.setItem = () => { throw Error('quota'); };
    const next = sample(); next.tabs[0].content = 'new unsaved';
    assert.equal(store.save(next), false); assert.equal(store.save(next), false);
    assert.equal(disk.getItem(store.key), old); assert.equal(warnings.length, 1);
    disk.setItem = write;
    assert.equal(store.save(next), true);
});
test('disabled storage warns rather than discarding silently', () => {
    const warnings = [];
    const store = createStore(() => { throw Error('denied'); }, null, message => warnings.push(message));
    assert.equal(store.load(), null); assert.equal(store.save(sample()), false);
    assert.equal(warnings.length, 1);
});
test('invalid view state and duplicate tab IDs fail validation', () => {
    const draft = sample(); draft.tabs[0].viewState = {};
    assert.throws(() => project(draft));
    draft.tabs[0].viewState = null; draft.tabs.push(draft.tabs[0]);
    assert.throws(() => project(draft));
});
test('menu paste has one owner, no native attempt or keybinding; denied access leaves keyboard paste available', async () => {
    let command; let reads = 0; const inserts = []; const warnings = [];
    const model = { getVersionId: () => 1, isDisposed: () => false };
    const editor = { getRawOptions: () => ({}), getModel: () => model, getSelections: () => [],
        focus() {}, trigger: (...args) => inserts.push(args) };
    const monaco = { editor: { registerCommand: (id, run) => {
        assert.equal(id, 'editor.action.clipboardPasteAction'); command = run; return { dispose() {} };
    } } };
    let read = async () => { reads++; return 'PASTE'; };
    installPaste(monaco, () => editor, () => ({ readText: () => read() }), message => warnings.push(message));
    await command(); assert.equal(reads, 1); assert.deepEqual(inserts, [['menu', 'paste', { text: 'PASTE' }]]);
    read = async () => { throw Error('denied'); };
    await command(); assert.equal(inserts.length, 1); assert.match(warnings[0], /Ctrl\/Cmd\+V/);
});
test('pending menu paste cannot leak into a different/changed model or run twice concurrently', async () => {
    let resolve; let version = 1; let inserts = 0; const warnings = [];
    const model = { getVersionId: () => version, isDisposed: () => false };
    const editor = { getRawOptions: () => ({}), getModel: () => model, getSelections: () => [],
        focus() {}, trigger: () => inserts++ };
    const paste = installPaste({ editor: { registerCommand: () => ({ dispose() {} }) } }, () => editor,
        () => ({ readText: () => new Promise(done => { resolve = done; }) }), message => warnings.push(message));
    const first = paste.run(); await paste.run(); version++; resolve('code'); await first;
    assert.equal(inserts, 0); assert.equal(warnings.length, 1);
});
