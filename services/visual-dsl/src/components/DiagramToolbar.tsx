import React, { useState, useEffect } from 'react';
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

  // Load saved diagrams list on component mount and after saves
  const loadSavedDiagrams = () => {
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
    
    // Sort by most recently modified
    diagrams.sort((a, b) => new Date(b.modified).getTime() - new Date(a.modified).getTime());
    setSavedDiagrams(diagrams);
  };

  useEffect(() => {
    loadSavedDiagrams();
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

  const loadDiagramFromStorage = (key: string) => {
    const storageData = localStorage.getItem(key);
    if (storageData) {
      try {
        const diagramData = JSON.parse(storageData);
        onLoad(diagramData);
        onDiagramNameChange(diagramData.name || 'Untitled Diagram');
        setIsDropdownOpen(false);
      } catch (error) {
        console.error('Error loading diagram:', error);
      }
    }
  };

  const deleteDiagram = (key: string, name: string) => {
    const confirmed = window.confirm(`Delete diagram "${name}"? This cannot be undone.`);
    if (confirmed) {
      localStorage.removeItem(key);
      
      // Remove from diagram list
      const diagramList = JSON.parse(localStorage.getItem('diagram_list') || '[]');
      const updatedList = diagramList.filter((k: string) => k !== key);
      localStorage.setItem('diagram_list', JSON.stringify(updatedList));
      
      loadSavedDiagrams();
    }
  };

  const duplicateDiagram = (key: string, name: string) => {
    const diagramData = localStorage.getItem(key);
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

  return (
    <>
      <div className="bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700 px-4 py-2">
        <div className="flex items-center gap-4">
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
          <div className="flex items-center gap-2">
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
              onClick={() => document.getElementById('file-upload')?.click()}
              className="h-8 text-xs px-3 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-300 dark:border-gray-600"
              title="Import diagram from JSON file"
            >
              📥 Import
            </Button>

            {/* Chariot Code Generator */}
            <ChariotCodeGeneratorComponent 
              diagramData={diagramData}
            />
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
    </>
  );
};
