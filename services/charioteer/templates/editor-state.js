/* Browser-local drafts. Explicit projection: never serialize editor/DOM/auth objects. */
(function (root, factory) {
    const api = factory();
    if (typeof module === 'object' && module.exports) module.exports = api;
    else root.CharioteerDraft = api;
})(typeof globalThis === 'object' ? globalThis : this, function () {
    'use strict';
    const tabStrings = ['id', 'scope', 'fileName', 'displayName', 'content', 'originalContent',
        'sourceType', 'diagramSourceName'];
    const tabBooleans = ['isDirty', 'loaded', 'isUntitled', 'pinnedInReference'];
    const strings = ['view', 'bottomTab', 'activeId', 'referenceId', 'browserScope', 'diagramScope',
        'standalone', 'standaloneOriginal', 'standaloneName'];
    const views = ['files', 'functions', 'diagrams', 'dashboard', 'agents', 'mcp'];
    const fieldIds = ['createPlanCode']; // Code only; arbitrary vars, payloads and connection forms may contain secrets.
    const viewportIds = ['runtimePanel', 'outputContent', 'problemsContent'];
    function viewState(value) {
        if (value == null) return null;
        // Monaco view state is JSON data, not a live model. Reject malformed shapes before applying it.
        if (typeof value !== 'object' || !Array.isArray(value.cursorState) ||
            typeof value.viewState !== 'object' || value.viewState === null) throw Error('Invalid editor view state');
        return JSON.parse(JSON.stringify(value));
    }
    function project(input) {
        if (!input || input.version !== 1 || !Array.isArray(input.tabs)) throw Error('Unsupported draft');
        const result = { version: 1, tabs: [], order: [], last: {}, fields: {}, layout: {}, viewport: {} };
        for (const key of strings) if (typeof input[key] === 'string') result[key] = input[key];
        if (!views.includes(result.view)) result.view = 'files';
        result.bottomTab = result.bottomTab === 'problems' ? 'problems' : 'output';
        const ids = new Set();
        for (const source of input.tabs) {
            if (!source || typeof source.id !== 'string' || !source.id || ids.has(source.id)) throw Error('Invalid draft tab');
            const tab = {};
            for (const key of tabStrings) if (typeof source[key] === 'string') tab[key] = source[key];
            for (const key of tabBooleans) tab[key] = !!source[key];
            if (typeof tab.content !== 'string' || typeof tab.originalContent !== 'string') throw Error('Missing draft code');
            tab.isDirty = tab.content !== tab.originalContent;
            tab.viewState = viewState(source.viewState);
            ids.add(tab.id);
            result.tabs.push(tab);
        }
        if (!Array.isArray(input.order)) throw Error('Invalid draft order');
        result.order = [...new Set(input.order.filter(id => ids.has(id)))];
        for (const tab of result.tabs) if (!result.order.includes(tab.id) && tab.id !== result.referenceId) result.order.push(tab.id);
        for (const view of ['files', 'functions', 'diagrams']) {
            if (ids.has(input.last?.[view])) result.last[view] = input.last[view];
        }
        for (const id of fieldIds) if (typeof input.fields?.[id] === 'string') result.fields[id] = input.fields[id];
        for (const [key, min, max] of [['leftWidth', 200, 600], ['bottomHeight', 100, 2000]]) {
            const n = input.layout?.[key];
            if (Number.isFinite(n)) result.layout[key] = Math.max(min, Math.min(max, n));
        }
        for (const id of viewportIds) if (Number.isFinite(input.viewport?.[id])) result.viewport[id] = Math.max(0, input.viewport[id]);
        result.streaming = input.streaming !== false;
        result.fileScopePinned = !!input.fileScopePinned;
        result.diagramScopePinned = !!input.diagramScopePinned;
        result.referenceView = viewState(input.referenceView);
        result.standaloneView = viewState(input.standaloneView);
        return result;
    }
    function createStore(storage, user, warn) {
        const key = 'charioteer.editor.draft.v1:' + JSON.stringify(user || null);
        let ready = false;
        let blocked = false;
        let last = null;
        let warned = false;
        function warning(message) {
            if (!warned) warn(message);
            warned = true;
        }
        return {
            key,
            load() {
                try {
                    const raw = storage().getItem(key);
                    const value = raw === null ? null : project(JSON.parse(raw));
                    last = raw;
                    ready = true;
                    return value;
                } catch (error) {
                    // Keep the original bytes for recovery; never autosave defaults over a corrupt/unknown draft.
                    blocked = true;
                    ready = true;
                    warning('Draft could not be restored. The stored copy is untouched; automatic draft saving is paused. Export your code before leaving.');
                    return null;
                }
            },
            save(input) {
                if (!ready || blocked) return false;
                try {
                    const raw = JSON.stringify(project(input));
                    if (raw !== last) storage().setItem(key, raw);
                    last = raw;
                    warned = false;
                    return true;
                } catch (error) {
                    warning('Draft could not be saved (browser storage unavailable or full). Export your code before leaving; the last saved copy is untouched.');
                    return false;
                }
            }
        };
    }
    function installPaste(monaco, getEditor, clipboard, warn) {
        // Keyboard paste belongs ONLY to Monaco's native textarea paste listener: no keybinding,
        // DOM paste listener, execCommand or asynchronous read is registered for Ctrl/Cmd+V.
        // Menu paste uses ONE async read. Monaco 0.45's execCommand(false) -> readText fallback
        // can insert twice when the host dispatches paste but reports false.
        let busy = false;
        const run = async () => {
            const editor = getEditor();
            if (busy || !editor || editor.getRawOptions().readOnly) return;
            const model = editor.getModel();
            const version = model.getVersionId();
            const selections = editor.getSelections();
            busy = true;
            try {
                const text = await clipboard().readText();
                if (editor.getModel() !== model || model.isDisposed() || model.getVersionId() !== version ||
                    JSON.stringify(editor.getSelections()) !== JSON.stringify(selections)) {
                    warn('Paste cancelled because the editor changed. Paste again in the intended tab.');
                    return;
                }
                editor.focus();
                editor.trigger('menu', 'paste', { text });
            } catch (error) {
                warn('Clipboard access denied. Focus the editor and use Ctrl/Cmd+V to paste.');
            } finally { busy = false; }
        };
        const registration = monaco.editor.registerCommand('editor.action.clipboardPasteAction', run);
        return { run, dispose: () => registration.dispose() };
    }
    return { project, createStore, installPaste, fieldIds, viewportIds };
});
