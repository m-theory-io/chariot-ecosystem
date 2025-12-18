import React, { useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

export interface CloneNodeProperties {
  object: string;
  newName: string;
}

interface CloneNodePropertiesDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: CloneNodeProperties) => void;
  onDelete: () => void;
  initialProperties?: CloneNodeProperties;
}

export const CloneNodePropertiesDialog: React.FC<CloneNodePropertiesDialogProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [object, setObject] = useState(initialProperties?.object || '');
  const [newName, setNewName] = useState(initialProperties?.newName || '');

  const handleSave = () => {
    onSave({ object, newName });
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
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Clone Properties
          </h3>
          <button
            onClick={handleSave}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
          >
            ×
          </button>
        </div>
        <div className="p-6">
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Object to Clone:
            </label>
            <Input
              type="text"
              value={object}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setObject(e.target.value)}
              className="w-full"
              placeholder="myArray"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Array, map, JSON, MapNode, TreeNode, or SimpleJSON
            </p>
          </div>
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              New Name (optional):
            </label>
            <Input
              type="text"
              value={newName}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setNewName(e.target.value)}
              className="w-full"
              placeholder="copyName"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              For nodes, provide a new name for the cloned node
            </p>
          </div>
          <div className="flex gap-3">
            <Button
              onClick={handleSave}
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
