import React, { useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface TreeSaveNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: TreeSaveNodeProperties) => void;
  onDelete: () => void;
  initialProperties: TreeSaveNodeProperties;
}

export interface TreeSaveNodeProperties {
  treeVariable: string;     // Symbolic name of TreeNode variable to be saved
  filename: string;         // Filename to save to, including file extension
  format?: string;          // Optional format specifier (json, gob)
  compress?: boolean;       // Optional compress flag
}

export const TreeSaveNodePropertiesDialog: React.FC<TreeSaveNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [treeVariable, setTreeVariable] = useState(initialProperties.treeVariable || '');
  const [filename, setFilename] = useState(initialProperties.filename || '');
  const [format, setFormat] = useState(initialProperties.format || 'json');
  const [compress, setCompress] = useState(initialProperties.compress || false);

  const handleSave = () => {
    onSave({
      treeVariable: treeVariable.trim(),
      filename: filename.trim(),
      format: format || undefined,
      compress: compress
    });
    onClose();
  };

  const handleClose = () => {
    // Save properties when closing
    onSave({
      treeVariable: treeVariable.trim(),
      filename: filename.trim(),
      format: format || undefined,
      compress: compress
    });
    onClose();
  };

  const handleDelete = () => {
    onDelete();
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-md w-full mx-4">
        {/* Title Bar */}
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Tree Save Properties
          </h3>
          <button
            onClick={handleClose}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
          >
            ×
          </button>
        </div>
        
        {/* Content */}
        <div className="p-6">
          {/* Tree Variable */}
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Tree Variable:
            </label>
            <Input
              type="text"
              value={treeVariable}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setTreeVariable(e.target.value)}
              className="w-full"
              placeholder="usersAgent"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Symbolic name of TreeNode variable to be saved
            </p>
          </div>

          {/* Filename */}
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Filename:
            </label>
            <Input
              type="text"
              value={filename}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setFilename(e.target.value)}
              className="w-full"
              placeholder="usersAgent.json"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Filename to save to, including file extension (.json, .gob, .xml, .yaml)
            </p>
          </div>

          {/* Format */}
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Format (optional):
            </label>
            <select
              value={format}
              onChange={(e: React.ChangeEvent<HTMLSelectElement>) => setFormat(e.target.value)}
              className="w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400"
            >
              <option value="">Auto-detect from extension</option>
              <option value="json">JSON</option>
              <option value="gob">GOB</option>
              <option value="xml">XML</option>
              <option value="yaml">YAML</option>
            </select>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Optional format specifier (json, gob, xml, yaml)
            </p>
          </div>

          {/* Compress */}
          <div className="mb-6">
            <label className="flex items-center space-x-2 text-sm font-medium text-gray-700 dark:text-gray-300">
              <input
                type="checkbox"
                checked={compress}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => setCompress(e.target.checked)}
                className="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500 dark:focus:ring-blue-400 dark:bg-gray-700"
              />
              <span>Enable compression</span>
            </label>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1 ml-6">
              Optional compress flag (true | false)
            </p>
          </div>
          
          {/* Buttons */}
          <div className="flex gap-3">
            <Button
              onClick={handleClose}
              className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200"
            >
              Save Properties
            </Button>
            <Button
              onClick={handleDelete}
              className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200"
            >
              Delete
            </Button>
          </div>
        </div>
      </div>
      
      {/* Explanatory text */}
      <div className="absolute top-1/2 left-1/2 transform translate-x-64 -translate-y-1/2 max-w-sm p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">
          The Tree Save logicon saves a TreeNode variable to a file with various format options.
        </p>
        <p className="text-sm text-gray-700 dark:text-gray-300 mt-2">
          <strong>Parameters:</strong>
        </p>
        <ul className="text-xs text-gray-600 dark:text-gray-400 mt-1 space-y-1">
          <li>1. Tree Variable - The TreeNode variable to save</li>
          <li>2. Filename - Target file with extension</li>
          <li>3. Format - Optional format (json, gob, xml, yaml)</li>
          <li>4. Compress - Optional compression flag</li>
        </ul>
        <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">
          💡 Examples:<br/>
          treeSave(tree, 'data.json')<br/>
          treeSave(tree, 'data.xml', 'xml')<br/>
          treeSave(tree, 'data.yaml', 'yaml', true)
        </p>
      </div>
    </div>
  );
};
