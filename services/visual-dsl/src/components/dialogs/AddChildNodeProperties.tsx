import React, { useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface AddChildNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: AddChildNodeProperties) => void;
  onDelete: () => void;
  initialProperties: AddChildNodeProperties;
}

export interface AddChildNodeProperties {
  parentNode: string; // The Node name to receive the child
  childNode: string;  // The Node object to add to the children
}

export const AddChildNodePropertiesDialog: React.FC<AddChildNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [parentNode, setParentNode] = useState(initialProperties.parentNode || '');
  const [childNode, setChildNode] = useState(initialProperties.childNode || '');

  const handleSave = () => {
    onSave({
      parentNode: parentNode.trim(),
      childNode: childNode.trim()
    });
    onClose();
  };

  const handleClose = () => {
    // Save properties when closing
    onSave({
      parentNode: parentNode.trim(),
      childNode: childNode.trim()
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
            Add Child Properties
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
          {/* Parent Node */}
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Parent Node:
            </label>
            <Input
              type="text"
              value={parentNode}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setParentNode(e.target.value)}
              className="w-full"
              placeholder="usersAgent"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              The Node name to receive the child
            </p>
          </div>

          {/* Child Node */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Child Node:
            </label>
            <Input
              type="text"
              value={childNode}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setChildNode(e.target.value)}
              className="w-full"
              placeholder="rules"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              The Node object to add to the children
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
          The Add Child logicon adds a child node to a parent node in the tree structure.
        </p>
        <p className="text-sm text-gray-700 dark:text-gray-300 mt-2">
          <strong>Parameters:</strong>
        </p>
        <ul className="text-xs text-gray-600 dark:text-gray-400 mt-1 space-y-1">
          <li>1. Parent Node - The node to receive the child</li>
          <li>2. Child Node - The node to be added as a child</li>
        </ul>
        <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">
          💡 Example: addChild(usersAgent, rules)
        </p>
      </div>
    </div>
  );
};
