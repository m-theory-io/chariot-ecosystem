import React, { useEffect, useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface FindByNameNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: FindByNameNodeProperties) => void;
  onDelete: () => void;
  initialProperties: FindByNameNodeProperties;
}

export interface FindByNameNodeProperties {
  node: string; // variable name of the node to search within
  name: string; // child name to find
}

export const FindByNameNodePropertiesDialog: React.FC<FindByNameNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [node, setNode] = useState(initialProperties.node || '');
  const [name, setName] = useState(initialProperties.name || '');
  const [canSave, setCanSave] = useState(false);

  useEffect(() => {
    setCanSave(node.trim().length > 0 && name.trim().length > 0);
  }, [node, name]);

  const handleSave = () => {
    if (!canSave) return;
    onSave({ node: node.trim(), name: name.trim() });
    onClose();
  };

  const handleClose = () => {
    if (canSave) {
      onSave({ node: node.trim(), name: name.trim() });
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
            Find By Name Properties
          </h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
          >
            ×
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-6">
          {/* Node variable (required) */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Node Variable (required):
            </label>
            <Input
              type="text"
              value={node}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setNode(e.target.value)}
              className="w-full"
              placeholder="e.g. users"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              The variable of the parent node to search within.
            </p>
          </div>

          {/* Child name (required) */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Child Name (required):
            </label>
            <Input
              type="text"
              value={name}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setName(e.target.value)}
              className="w-full"
              placeholder="e.g. Alice"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              The name of the child node to find.
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
          Find By Name searches the children of a node by exact name.
        </p>
        <p className="text-sm text-gray-700 dark:text-gray-300 mt-2">
          <strong>Parameters:</strong>
        </p>
        <ul className="text-xs text-gray-600 dark:text-gray-400 mt-1 space-y-1">
          <li>1. node (TreeNode, required)</li>
          <li>2. name (string, required)</li>
        </ul>
        <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">
          💡 Returns the matched child or DBNull if not found.
        </p>
      </div>
    </div>
  );
};
