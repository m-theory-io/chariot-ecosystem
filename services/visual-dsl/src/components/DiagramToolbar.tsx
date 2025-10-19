import React, { useState, useEffect, useRef } from 'react';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { ChariotCodeGeneratorComponent } from './ChariotCodeGenerator';

interface DiagramToolbarProps {
  currentDiagramName: string;
  onDiagramNameChange: (name: string) => void;
  onNew: () => void;
  onSave: () => void;
  onLoad: (jsonData: string) => void;
  onExport: () => void;
  diagramData?: any; // Added for Chariot code generation
}

interface SavedDiagram {
  key: string;
  name: string;
  modified: string;
}

export const DiagramToolbar: React.FC<DiagramToolbarProps> = ({
  currentDiagramName,
  onDiagramNameChange,
  onNew,
  onSave,
  onLoad,
  onExport,
  diagramData
}) => {
  const [isLoadDialogOpen, setIsLoadDialogOpen] = useState(false);
  const [loadJsonText, setLoadJsonText] = useState('');
  const [savedDiagrams, setSavedDiagrams] = useState<SavedDiagram[]>([]);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [showSaveAsDialog, setShowSaveAsDialog] = useState(false);
  const [saveAsName, setSaveAsName] = useState('');
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [isServerImportOpen, setIsServerImportOpen] = useState(false);
  const [saveDirectory, setSaveDirectory] = useState('');
  const [pendingSaveDirectory, setPendingSaveDirectory] = useState('');
  // Storage mode: local folder (default) or server API
  const [storageMode, setStorageMode] = useState<'local' | 'server'>('local');
  const [serverBaseUrl, setServerBaseUrl] = useState('');
  const [pendingStorageMode, setPendingStorageMode] = useState<'local' | 'server'>('local');
  const [pendingServerBaseUrl, setPendingServerBaseUrl] = useState('');
  const [testConnStatus, setTestConnStatus] = useState<'idle' | 'testing' | 'ok' | 'error'>('idle');
  const [testConnMessage, setTestConnMessage] = useState<string>('');
  const getAuthToken = () => localStorage.getItem('chariot_auth_token') || '';
  const setAuthToken = (t: string) => localStorage.setItem('chariot_auth_token', t);
  const directoryInputRef = useRef<HTMLInputElement | null>(null);

  // Load saved diagrams list (local or server based on storage mode)
  const loadSavedDiagrams = async () => {
    if (storageMode === 'server') {
      try {
        const token = localStorage.getItem('chariot_auth_token') || '';
        const res = await fetch(serverBaseUrl, { credentials: 'include', headers: token ? { Authorization: token } : undefined });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const payload = await res.json();
        const list = Array.isArray(payload?.Data)
          ? payload.Data
          : Array.isArray(payload?.data)
          ? payload.data
          : [];
        const diagrams: SavedDiagram[] = list.map((d: any) => ({
          key: d.Name || d.name,
          name: (d.Name || d.name) ?? 'Untitled',
          modified: (d.Modified || d.modified) ? new Date(d.Modified || d.modified).toISOString() : new Date().toISOString(),
        }));
        diagrams.sort((a, b) => new Date(b.modified).getTime() - new Date(a.modified).getTime());
        setSavedDiagrams(diagrams);
      } catch (err) {
        console.warn('Failed to fetch server diagrams list:', err);
        setSavedDiagrams([]);
      }
      return;
    }

    // Local storage fallback
    const diagramList = JSON.parse(localStorage.getItem('diagram_list') || '[]');
    const diagrams: SavedDiagram[] = [];
    diagramList.forEach((key: string) => {
      const diagramData = localStorage.getItem(key);
      if (diagramData) {
        try {
          const parsed = JSON.parse(diagramData);
          diagrams.push({
            key,
            name: parsed.name || 'Untitled',
            modified: parsed.modified || parsed.created || new Date().toISOString()
          });
        } catch (error) {
          // Skip invalid diagram data
        }
      }
    });
    diagrams.sort((a, b) => new Date(b.modified).getTime() - new Date(a.modified).getTime());
    setSavedDiagrams(diagrams);
  };

  useEffect(() => {
    // initial
    loadSavedDiagrams();
    const storedPath = localStorage.getItem('diagram_save_path') || '';
    setSaveDirectory(storedPath);
    const storedMode = (localStorage.getItem('diagram_storage_mode') as 'local' | 'server') || 'local';
    setStorageMode(storedMode);
    const defaultServer = '/api/diagrams';
    const storedServer = localStorage.getItem('diagram_server_base_url') || defaultServer;
    setServerBaseUrl(storedServer);
  }, []);

  // Refresh list when mode/base URL changes
  useEffect(() => {
    loadSavedDiagrams();
  }, [storageMode, serverBaseUrl]);

  useEffect(() => {
    if (directoryInputRef.current) {
      directoryInputRef.current.setAttribute('webkitdirectory', '');
      directoryInputRef.current.setAttribute('directory', '');
    }
  }, []);

  const handleSaveWithRefresh = () => {
    onSave();
    // Refresh the diagram list after save
    setTimeout(loadSavedDiagrams, 100);
  };

  const handleSaveAs = () => {
    setSaveAsName(currentDiagramName + ' Copy');
    setShowSaveAsDialog(true);
  };

  const confirmSaveAs = () => {
    if (saveAsName.trim()) {
      const originalName = currentDiagramName;
      onDiagramNameChange(saveAsName.trim());
      
      // Small delay to ensure name is updated before save
      setTimeout(() => {
        onSave();
        setTimeout(loadSavedDiagrams, 100);
      }, 50);
      
      setShowSaveAsDialog(false);
      setSaveAsName('');
    }
  };

  const loadDiagramFromStorage = async (key: string) => {
    if (storageMode === 'server') {
      try {
        const url = `${serverBaseUrl.replace(/\/$/, '')}/${encodeURIComponent(key)}`;
        const token = localStorage.getItem('chariot_auth_token') || '';
        const res = await fetch(url, { credentials: 'include', headers: token ? { Authorization: token } : undefined });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const text = await res.text();
        onLoad(text);
        onDiagramNameChange(key);
        setIsDropdownOpen(false);
      } catch (err) {
        alert('Failed to load diagram from server.');
      }
      return;
    }
    const storageData = localStorage.getItem(key);
    if (storageData) {
      // Always pass the raw JSON string to the loader; it will normalize/parse safely
      onLoad(storageData);
      // Try to parse locally just to update the name, but don't block on failure
      try {
        const parsed = JSON.parse(storageData);
        onDiagramNameChange(parsed.name || 'Untitled Diagram');
      } catch (error) {
        console.warn('Saved diagram name parse failed; using key as fallback');
        try {
          onDiagramNameChange(key.replace(/^diagram_/, ''));
        } catch (_) {}
      }
      setIsDropdownOpen(false);
    }
  };

  const deleteDiagram = async (key: string, name: string) => {
    const confirmed = window.confirm(`Delete diagram "${name}"? This cannot be undone.`);
    if (confirmed) {
      if (storageMode === 'server') {
        try {
          const url = `${serverBaseUrl.replace(/\/$/, '')}/${encodeURIComponent(key)}`;
          const token = localStorage.getItem('chariot_auth_token') || '';
          const res = await fetch(url, { method: 'DELETE', credentials: 'include', headers: token ? { Authorization: token } : undefined });
          if (!res.ok && res.status !== 204) throw new Error(`HTTP ${res.status}`);
        } catch (err) {
          alert('Failed to delete diagram on server.');
          return;
        }
      } else {
        localStorage.removeItem(key);
        const diagramList = JSON.parse(localStorage.getItem('diagram_list') || '[]');
        const updatedList = diagramList.filter((k: string) => k !== key);
        localStorage.setItem('diagram_list', JSON.stringify(updatedList));
      }
      loadSavedDiagrams();
    }
  };

  const duplicateDiagram = async (key: string, name: string) => {
    const diagramData = localStorage.getItem(key);
    if (storageMode === 'server') {
      try {
        // fetch original
        const url = `${serverBaseUrl.replace(/\/$/, '')}/${encodeURIComponent(key)}`;
        const token = localStorage.getItem('chariot_auth_token') || '';
        const res = await fetch(url, { credentials: 'include', headers: token ? { Authorization: token } : undefined });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const original = await res.json();
        const newName = `${name} Copy`;
        const saveRes = await fetch(serverBaseUrl, {
          method: 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: token } : {}) },
          body: JSON.stringify({ name: newName, content: original }),
        });
        if (!saveRes.ok && saveRes.status !== 204) throw new Error(`HTTP ${saveRes.status}`);
        loadSavedDiagrams();
      } catch (err) {
        alert('Failed to duplicate diagram on server.');
      }
      return;
    }
    if (diagramData) {
      try {
        const parsed = JSON.parse(diagramData);
        const newName = `${name} Copy`;
        const newKey = `diagram_${newName.replace(/[^a-zA-Z0-9]/g, '_')}`;
        
        const duplicatedData = {
          ...parsed,
          name: newName,
          created: new Date().toISOString(),
          modified: new Date().toISOString()
        };
        
        localStorage.setItem(newKey, JSON.stringify(duplicatedData));
        
        // Add to diagram list
        const diagramList = JSON.parse(localStorage.getItem('diagram_list') || '[]');
        if (!diagramList.includes(newKey)) {
          diagramList.push(newKey);
          localStorage.setItem('diagram_list', JSON.stringify(diagramList));
        }
        
        loadSavedDiagrams();
      } catch (error) {
        alert('Failed to duplicate diagram.');
      }
    }
  };

  const handleLoadClick = () => {
    setIsLoadDialogOpen(true);
    setLoadJsonText('');
  };

  const handleLoadConfirm = () => {
    if (loadJsonText.trim()) {
      try {
        // Validate JSON before passing it up
        JSON.parse(loadJsonText);
        onLoad(loadJsonText);
        setIsLoadDialogOpen(false);
        setLoadJsonText('');
      } catch (error) {
        alert('Invalid JSON format. Please check your input.');
      }
    }
  };

  const openSettings = () => {
    setPendingSaveDirectory(saveDirectory);
    setPendingStorageMode(storageMode);
    setPendingServerBaseUrl(serverBaseUrl);
    setTestConnStatus('idle');
    setTestConnMessage('');
    setIsSettingsOpen(true);
  };

  const closeSettings = () => {
    setIsSettingsOpen(false);
  };

  const saveSettings = () => {
    const trimmedPath = pendingSaveDirectory.trim();
    setSaveDirectory(trimmedPath);
    localStorage.setItem('diagram_save_path', trimmedPath);
    // Persist storage mode and server URL
    setStorageMode(pendingStorageMode);
    localStorage.setItem('diagram_storage_mode', pendingStorageMode);
    const trimmedServer = pendingServerBaseUrl.trim() || '/api/diagrams';
    setServerBaseUrl(trimmedServer);
    localStorage.setItem('diagram_server_base_url', trimmedServer);
    setIsSettingsOpen(false);
  };

  const testServerConnection = async () => {
    if (!pendingServerBaseUrl.trim()) return;
    setTestConnStatus('testing');
    setTestConnMessage('');
    try {
      const url = pendingServerBaseUrl.replace(/\/$/, '');
      const token = getAuthToken();
      const res = await fetch(url, { 
        credentials: 'include', 
        headers: token ? { Authorization: token } : undefined,
      });
      if (!res.ok) {
        if (res.status === 401) {
          // Try auto-login for dev
          try {
            const loginRes = await fetch('/login', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              credentials: 'include',
              body: JSON.stringify({ username: 'dev', password: 'dev' }),
            });
            if (loginRes.ok) {
              const body = await loginRes.json();
              const t = body?.Data?.token || body?.data?.token;
              if (t) {
                setAuthToken(t);
                // retry ping with token
                const retry = await fetch(url, { credentials: 'include', headers: { Authorization: t } });
                if (retry.ok) {
                  setTestConnStatus('ok');
                  setTestConnMessage('Connected');
                  return;
                } else {
                  setTestConnStatus('error');
                  setTestConnMessage(`HTTP ${retry.status}`);
                  return;
                }
              }
            }
          } catch (e) {
            // fallthrough to error below
          }
        }
        setTestConnStatus('error');
        setTestConnMessage(`HTTP ${res.status}`);
        return;
      }
      // Expect ResultJSON: { Result: 'OK', Data: [...] }
      const body = await res.json();
      const result = body?.Result || body?.result || '';
      if (result && String(result).toUpperCase() !== 'OK') {
        setTestConnStatus('error');
        setTestConnMessage(typeof body?.Data === 'string' ? body.Data : 'Unexpected response');
        return;
      }
      setTestConnStatus('ok');
      setTestConnMessage('Connected');
    } catch (err: any) {
      setTestConnStatus('error');
      setTestConnMessage(err?.message || 'Network error');
    }
  };

  const handleDirectorySelection = async () => {
    try {
      if (typeof window !== 'undefined' && typeof (window as any).showDirectoryPicker === 'function') {
        const handle = await (window as any).showDirectoryPicker();
        if (!handle) {
          return;
        }

        if (typeof (handle as any).resolve === 'function') {
          const segments = await (handle as any).resolve();
          if (Array.isArray(segments) && segments.length > 0) {
            setPendingSaveDirectory('/' + segments.join('/'));
            return;
          }
        }

        const legacyPath = (handle as any).fullPath || (handle as any).path;
        if (legacyPath) {
          setPendingSaveDirectory(legacyPath);
        } else if (handle.name) {
          if (window.location.origin === 'http://localhost') {
            setPendingSaveDirectory(`/Volumes/${handle.name}`);
          } else {
            setPendingSaveDirectory(handle.name);
          }
        }
      } else if (directoryInputRef.current) {
        directoryInputRef.current.click();
      }
    } catch (error) {
      console.warn('Directory selection was cancelled or failed:', error);
    }
  };

  const handleDirectoryInputChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = event.target.files;
    if (files && files.length > 0) {
      const firstFile = files[0] as File & { webkitRelativePath?: string };
      if (firstFile.webkitRelativePath) {
        const segments = firstFile.webkitRelativePath.split('/');
        if (segments.length > 0) {
          segments.pop();
        }
        const basePath = (firstFile as any).path || '';
        if (basePath) {
          const normalized = basePath.replace(/\\/g, '/');
          const lastSlash = normalized.lastIndexOf('/');
          const withoutFile = lastSlash >= 0 ? normalized.substring(0, lastSlash) : normalized;
          setPendingSaveDirectory(withoutFile || normalized);
        } else {
          setPendingSaveDirectory('/' + segments.join('/'));
        }
      } else if ((firstFile as any).path) {
        const raw = (firstFile as any).path;
        const normalized = raw.replace(/\\/g, '/');
        const lastSlash = normalized.lastIndexOf('/');
        const withoutFile = lastSlash >= 0 ? normalized.substring(0, lastSlash) : normalized;
        setPendingSaveDirectory(withoutFile || normalized);
      } else {
        setPendingSaveDirectory(firstFile.name);
      }
    }
    event.target.value = '';
  };

  const handleFileUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (e) => {
        const content = e.target?.result as string;
        if (content) {
          try {
            JSON.parse(content); // Validate JSON
            onLoad(content);
          } catch (error) {
            alert('Invalid JSON file format.');
          }
        }
      };
      reader.readAsText(file);
    }
    // Reset the input
    event.target.value = '';
  };

  // Import: if storage is server, open a server picker; else fall back to file upload
  const handleImportClick = async () => {
    if (storageMode === 'server') {
      await loadSavedDiagrams();
      setIsServerImportOpen(true);
      return;
    }
    document.getElementById('file-upload')?.click();
  };

  const handleServerImport = async (key: string) => {
    await loadDiagramFromStorage(key);
    setIsServerImportOpen(false);
  };

  return (
    <>
      <div className="bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700 px-4 py-2">
        <div className="flex flex-wrap items-center gap-4">
          {/* Diagram Name */}
          <div className="flex items-center gap-2">
            <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
              Diagram:
            </label>
            <Input
              type="text"
              value={currentDiagramName}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => onDiagramNameChange(e.target.value)}
              placeholder="Untitled Diagram"
              className="w-48 h-8 text-sm"
            />
          </div>

          {/* File Operations */}
          <div className="flex items-center gap-2 flex-wrap flex-1">
            <Button
              onClick={onNew}
              className="h-8 text-xs px-3 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-300 dark:border-gray-600"
              title="Create new diagram"
            >
              📄 New
            </Button>

            <Button
              onClick={handleSaveWithRefresh}
              className="h-8 text-xs px-3 bg-green-100 hover:bg-green-200 dark:bg-green-700 dark:hover:bg-green-600 text-green-800 dark:text-green-200 border border-green-300 dark:border-green-600"
              title="Save diagram"
            >
              💾 Save
            </Button>

            <Button
              onClick={handleSaveAs}
              className="h-8 text-xs px-3 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-300 dark:border-gray-600"
              title="Save diagram with new name"
            >
              📋 Save As
            </Button>

            {/* Saved Diagrams Dropdown */}
            <div className="relative">
              <Button
                onClick={() => setIsDropdownOpen(!isDropdownOpen)}
                className="h-8 text-xs px-3 bg-blue-100 hover:bg-blue-200 dark:bg-blue-700 dark:hover:bg-blue-600 text-blue-800 dark:text-blue-200 border border-blue-300 dark:border-blue-600"
                title="Open saved diagram"
              >
                📂 Open {savedDiagrams.length > 0 && `(${savedDiagrams.length})`}
              </Button>
              
              {isDropdownOpen && (
                <div className="absolute top-full left-0 mt-1 w-80 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-md shadow-lg z-50 max-h-64 overflow-y-auto">
                  {savedDiagrams.length === 0 ? (
                    <div className="p-3 text-sm text-gray-500 dark:text-gray-400">No saved diagrams</div>
                  ) : (
                    savedDiagrams.map((diagram) => (
                      <div key={diagram.key} className="border-b border-gray-100 dark:border-gray-700 last:border-b-0">
                        <div className="p-2 hover:bg-gray-50 dark:hover:bg-gray-700">
                          <div className="flex items-center justify-between">
                            <button
                              onClick={() => loadDiagramFromStorage(diagram.key)}
                              className="flex-1 text-left text-sm font-medium text-gray-900 dark:text-gray-100 hover:text-blue-600 dark:hover:text-blue-400"
                            >
                              {diagram.name}
                            </button>
                            <div className="flex items-center gap-1 ml-2">
                              <button
                                onClick={() => duplicateDiagram(diagram.key, diagram.name)}
                                className="p-1 text-xs text-gray-500 hover:text-blue-600 dark:text-gray-400 dark:hover:text-blue-400"
                                title="Duplicate"
                              >
                                📋
                              </button>
                              <button
                                onClick={() => deleteDiagram(diagram.key, diagram.name)}
                                className="p-1 text-xs text-gray-500 hover:text-red-600 dark:text-gray-400 dark:hover:text-red-400"
                                title="Delete"
                              >
                                �️
                              </button>
                            </div>
                          </div>
                          <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            Modified: {new Date(diagram.modified).toLocaleDateString()} {new Date(diagram.modified).toLocaleTimeString()}
                          </div>
                        </div>
                      </div>
                    ))
                  )}
                </div>
              )}
              
              {/* Click outside to close dropdown */}
              {isDropdownOpen && (
                <div 
                  className="fixed inset-0 z-40" 
                  onClick={() => setIsDropdownOpen(false)}
                ></div>
              )}
            </div>

            <Button
              onClick={handleLoadClick}
              className="h-8 text-xs px-3 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-300 dark:border-gray-600"
              title="Load diagram from JSON"
            >
              � Paste JSON
            </Button>

            <Button
              onClick={onExport}
              className="h-8 text-xs px-3 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-300 dark:border-gray-600"
              title="Export diagram as JSON file"
            >
              📤 Export
            </Button>

            {/* Hidden file input for loading from file */}
            <input
              type="file"
              accept=".json"
              onChange={handleFileUpload}
              style={{ display: 'none' }}
              id="file-upload"
            />
            <Button
              onClick={handleImportClick}
              className="h-8 text-xs px-3 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-300 dark:border-gray-600"
              title={storageMode === 'server' ? 'Import diagram from server' : 'Import diagram from JSON file'}
            >
              {storageMode === 'server' ? '📥 Import (Server)' : '📥 Import'}
            </Button>

            {/* Chariot Code Generator */}
            <ChariotCodeGeneratorComponent 
              diagramData={diagramData}
            />

            <div className="flex items-center gap-2 ml-auto">
              <span className="text-xs text-gray-500 dark:text-gray-400 whitespace-nowrap">
                {storageMode === 'local'
                  ? (saveDirectory ? `Save Path: ${saveDirectory}` : 'Storage: Local')
                  : `Storage: Server (${serverBaseUrl})`}
              </span>
              <Button
                onClick={openSettings}
                className="h-8 text-xs px-3 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-300 dark:border-gray-600"
                title="Edit toolbar settings"
              >
                ⚙️ Settings
              </Button>
            </div>
          </div>
        </div>
      </div>

      {/* Save As Dialog */}
      {showSaveAsDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">
              Save As New Diagram
            </h3>
            <Input
              type="text"
              value={saveAsName}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSaveAsName(e.target.value)}
              placeholder="Enter diagram name"
              className="w-full mb-4"
            />
            <div className="flex justify-end gap-2">
              <Button
                onClick={() => setShowSaveAsDialog(false)}
                className="px-4 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-300 dark:border-gray-600"
              >
                Cancel
              </Button>
              <Button
                onClick={confirmSaveAs}
                className={`px-4 py-2 ${!saveAsName.trim() 
                  ? 'bg-gray-300 dark:bg-gray-600 text-gray-500 cursor-not-allowed' 
                  : 'bg-green-600 hover:bg-green-700 dark:bg-green-500 dark:hover:bg-green-600 text-white'
                }`}
              >
                Save As
              </Button>
            </div>
          </div>
        </div>
      )}

      {isSettingsOpen && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-3xl w-full mx-4">
            <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">
              Settings
            </h3>
            <div className="space-y-4">
              {/* Storage Mode */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Storage Mode
                </label>
                <div className="flex gap-2">
                  <button
                    className={`px-3 py-1 text-sm rounded border ${pendingStorageMode === 'local' ? 'bg-blue-600 text-white border-blue-600' : 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-200 border-gray-300 dark:border-gray-600'}`}
                    onClick={() => setPendingStorageMode('local')}
                  >
                    Local
                  </button>
                  <button
                    className={`px-3 py-1 text-sm rounded border ${pendingStorageMode === 'server' ? 'bg-blue-600 text-white border-blue-600' : 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-200 border-gray-300 dark:border-gray-600'}`}
                    onClick={() => setPendingStorageMode('server')}
                  >
                    Server
                  </button>
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  Choose where to save and open diagrams. Server mode uses your backend endpoints.
                </p>
              </div>

              {pendingStorageMode === 'local' ? (
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Diagram Save Path
                </label>
                <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                  <Input
                    type="text"
                    value={pendingSaveDirectory}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => setPendingSaveDirectory(e.target.value)}
                    placeholder="e.g., ~/Documents/ChariotDiagrams"
                    className="flex-1"
                  />
                  <Button
                    onClick={handleDirectorySelection}
                    className="h-8 text-xs px-3 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-300 dark:border-gray-600 whitespace-nowrap"
                  >
                    Select Folder
                  </Button>
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  Select a folder or enter a path manually. Stored locally for export helpers.
                </p>
                <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">
                  Tip: Some browsers label the confirmation button "Upload" when granting folder access. No files are uploaded—this simply authorizes reading from the chosen directory.
                </p>
              </div>
              ) : (
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Server Base URL
                </label>
                <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                  <Input
                    type="text"
                    value={pendingServerBaseUrl}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => setPendingServerBaseUrl(e.target.value)}
                    placeholder="e.g., https://your-host/api/diagrams"
                    className="flex-1"
                  />
                  <Button
                    onClick={testServerConnection}
                    className={`h-8 text-xs px-3 border whitespace-nowrap ${testConnStatus === 'testing' ? 'bg-gray-300 dark:bg-gray-600 text-gray-600' : 'bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border-gray-300 dark:border-gray-600'}`}
                    disabled={testConnStatus === 'testing'}
                    title="Ping the server URL"
                  >
                    {testConnStatus === 'testing' ? 'Testing…' : 'Test Connection'}
                  </Button>
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  Used for Save/Open when Storage Mode is Server. Defaults to your site origin at /api/diagrams.
                </p>
                {testConnStatus !== 'idle' && (
                  <p className={`text-xs mt-1 ${testConnStatus === 'ok' ? 'text-green-600' : 'text-red-600'}`}>
                    {testConnStatus === 'ok' ? '✓ ' : '✗ '} {testConnMessage}
                  </p>
                )}
              </div>
              )}
            </div>
            <div className="flex justify-end gap-2 mt-6">
              <Button
                onClick={closeSettings}
                className="px-4 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-300 dark:border-gray-600"
              >
                Cancel
              </Button>
              <Button
                onClick={saveSettings}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 text-white"
              >
                Save
              </Button>
            </div>
          </div>
        </div>
      )}

      <input
        ref={directoryInputRef}
        type="file"
        style={{ display: 'none' }}
        multiple
        onChange={handleDirectoryInputChange}
      />

      {/* Load JSON Dialog */}
      {isLoadDialogOpen && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-2xl w-full mx-4">
            <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">
              Load Diagram from JSON
            </h3>
            <textarea
              value={loadJsonText}
              onChange={(e) => setLoadJsonText(e.target.value)}
              placeholder="Paste your diagram JSON here..."
              className="w-full h-64 p-3 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 font-mono text-sm"
            />
            <div className="flex justify-end gap-2 mt-4">
              <Button
                onClick={() => setIsLoadDialogOpen(false)}
                className="px-4 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-300 dark:border-gray-600"
              >
                Cancel
              </Button>
              <Button
                onClick={handleLoadConfirm}
                className={`px-4 py-2 ${!loadJsonText.trim() 
                  ? 'bg-gray-300 dark:bg-gray-600 text-gray-500 cursor-not-allowed' 
                  : 'bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 text-white'
                }`}
              >
                Load Diagram
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Import from Server Dialog */}
      {isServerImportOpen && storageMode === 'server' && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-xl w-full mx-4">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Import Diagram from Server</h3>
              <span className="text-xs text-gray-500 dark:text-gray-400">{serverBaseUrl}</span>
            </div>
            <div className="border border-gray-200 dark:border-gray-700 rounded-md max-h-72 overflow-y-auto">
              {savedDiagrams.length === 0 ? (
                <div className="p-4 text-sm text-gray-500 dark:text-gray-400">No diagrams found on server.</div>
              ) : (
                <ul>
                  {savedDiagrams.map((d) => (
                    <li key={d.key} className="border-b last:border-b-0 border-gray-100 dark:border-gray-700">
                      <button
                        className="w-full text-left p-3 hover:bg-gray-50 dark:hover:bg-gray-700"
                        onClick={() => handleServerImport(d.key)}
                        title={`Import ${d.name}`}
                      >
                        <div className="flex items-center justify-between">
                          <span className="text-sm text-gray-900 dark:text-gray-100">{d.name}</span>
                          <span className="text-xs text-gray-500 dark:text-gray-400">{new Date(d.modified).toLocaleString()}</span>
                        </div>
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
            <div className="flex justify-between items-center mt-4">
              <Button
                onClick={() => loadSavedDiagrams()}
                className="px-3 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-300 dark:border-gray-600"
              >
                Refresh List
              </Button>
              <div className="flex gap-2">
                <Button
                  onClick={() => setIsServerImportOpen(false)}
                  className="px-4 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-300 dark:border-gray-600"
                >
                  Cancel
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  );
};
