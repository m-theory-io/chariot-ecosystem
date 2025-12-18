import React, { useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface CSVGetCellNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: CSVGetCellNodeProperties) => void;
  onDelete: () => void;
  initialProperties: CSVGetCellNodeProperties;
}

export interface CSVGetCellNodeProperties {
  nodeOrPath: string;
  rowIndex: string;
  colIndexOrName: string;
}

export const CSVGetCellNodePropertiesDialog: React.FC<CSVGetCellNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [nodeOrPath, setNodeOrPath] = useState(initialProperties.nodeOrPath || 'csvNode');
  const [rowIndex, setRowIndex] = useState(initialProperties.rowIndex || '0');
  const [colIndexOrName, setColIndexOrName] = useState(initialProperties.colIndexOrName || '0');

  const handleSave = () => {
    onSave({
      nodeOrPath,
      rowIndex,
      colIndexOrName
    });
    onClose();
  };

  const handleClose = () => {
    // Save properties when closing
    onSave({
      nodeOrPath,
      rowIndex,
      colIndexOrName
    });
    onClose();
  };



  const handleCancel = () => {
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
            CSV Get Cell Properties
          </h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
          >
            ×
          </button>
        </div>
        
        {/* Content */}
        <div className="p-6 space-y-4">
          {/* Info Box */}
          <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded p-3">
            <p className="text-sm text-blue-800 dark:text-blue-200">
              Get the value of a specific cell by row and column index or name.
            </p>
          </div>

          {/* Node or Path */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              CSV Node or File Path:
            </label>
            <Input
              type="text"
              value={nodeOrPath}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setNodeOrPath(e.target.value)}
              className="w-full"
              placeholder="csvNode or 'data/file.csv'"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              CSVNode variable name or file path string (in quotes)
            </p>
          </div>

          {/* Row Index */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Row Index:
            </label>
            <Input
              type="text"
              value={rowIndex}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setRowIndex(e.target.value)}
              className="w-full"
              placeholder="0"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Row index (0-based, excluding header row)
            </p>
          </div>

          {/* Column Index or Name */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Column Index or Name:
            </label>
            <Input
              type="text"
              value={colIndexOrName}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setColIndexOrName(e.target.value)}
              className="w-full"
              placeholder="0 or 'columnName'"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Column index (0-based number) or column name (string in quotes)
            </p>
          </div>
          
          {/* Buttons */}
          <div className="flex gap-3 pt-2">
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
    </div>
  );
};
