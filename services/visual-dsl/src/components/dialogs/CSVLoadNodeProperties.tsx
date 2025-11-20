import React, { useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface CSVLoadNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: CSVLoadNodeProperties) => void;
  onDelete: () => void;
  initialProperties: CSVLoadNodeProperties;
}

export interface CSVLoadNodeProperties {
  node: string;
  path: string;
}

export const CSVLoadNodePropertiesDialog: React.FC<CSVLoadNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [node, setNode] = useState(initialProperties.node || 'csvNode');
  const [path, setPath] = useState(initialProperties.path || 'data/file.csv');

  const handleSave = () => {
    onSave({
      node,
      path
    });
    onClose();
  };

  const handleClose = () => {
    // Save properties when closing
    onSave({
      node,
      path
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
            CSV Load Properties
          </h3>
          <button
            onClick={handleClose}
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
              Load a CSV file into an existing CSVNode instance.
            </p>
          </div>

          {/* CSV Node */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              CSV Node:
            </label>
            <Input
              type="text"
              value={node}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setNode(e.target.value)}
              className="w-full"
              placeholder="csvNode"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              CSVNode variable name
            </p>
          </div>

          {/* File Path */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              File Path:
            </label>
            <Input
              type="text"
              value={path}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setPath(e.target.value)}
              className="w-full"
              placeholder="data/file.csv"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Path to the CSV file to load
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
