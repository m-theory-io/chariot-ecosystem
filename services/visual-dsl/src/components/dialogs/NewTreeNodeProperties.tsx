import React, { useState, useEffect } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface NewTreeNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: NewTreeNodeProperties) => void;
  onDelete: () => void;
  initialProperties: NewTreeNodeProperties;
}

export interface NewTreeNodeProperties {
  nodeName: string;
}

export const NewTreeNodePropertiesDialog: React.FC<NewTreeNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [nodeName, setNodeName] = useState(initialProperties.nodeName || '');
  const [canSave, setCanSave] = useState(false);

  useEffect(() => {
    setCanSave(nodeName.trim().length > 0);
  }, [nodeName]);

  const handleSave = () => {
    if (!canSave) return;
    onSave({ nodeName: nodeName.trim() });
    onClose();
  };

  const handleClose = () => {
    // Only save if valid, otherwise just close without persisting invalid state
    if (canSave) {
      onSave({ nodeName: nodeName.trim() });
    }
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
            New Tree Properties
          </h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
          >
            ×
          </button>
        </div>
        
        {/* Content */}
        <div className="p-6">
          {/* Node Name (required) */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              TreeNode Name (required):
            </label>
            <Input
              type="text"
              value={nodeName}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setNodeName(e.target.value)}
              className="w-full"
              placeholder="e.g. MyTree"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              You must provide a name. This emits newTree('name').
            </p>
          </div>
          
          {/* Buttons */}
          <div className="flex gap-3">
            <Button
              onClick={handleSave}
              disabled={!canSave}
              className={`px-6 py-2 ${canSave ? 'bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600' : 'bg-gray-200 dark:bg-gray-600 opacity-60 cursor-not-allowed'} text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200`}
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
          The New Tree logicon creates a new Tree with the required name.
        </p>
        <p className="text-sm text-gray-700 dark:text-gray-300 mt-2">
          <strong>Parameter:</strong>
        </p>
        <ul className="text-xs text-gray-600 dark:text-gray-400 mt-1 space-y-1">
          <li>1. TreeNode name (string, required)</li>
        </ul>
        <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">
          💡 Save is disabled until a non-empty name is provided.
        </p>
      </div>
    </div>
  );
};
