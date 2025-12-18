import React, { useEffect, useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface GetChildAtNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: GetChildAtNodeProperties) => void;
  onDelete: () => void;
  initialProperties: GetChildAtNodeProperties;
}

export interface GetChildAtNodeProperties {
  node: string; // variable name of the node to get child from
  index: number; // index of the child (0-based)
}

export const GetChildAtNodePropertiesDialog: React.FC<GetChildAtNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [node, setNode] = useState(initialProperties.node || '');
  const [index, setIndex] = useState<number>(
    typeof initialProperties.index === 'number' ? initialProperties.index : 0
  );
  const [canSave, setCanSave] = useState(false);

  useEffect(() => {
    const validNode = node.trim().length > 0;
    const validIndex = Number.isInteger(index) && index >= 0;
    setCanSave(validNode && validIndex);
  }, [node, index]);

  const handleSave = () => {
    if (!canSave) return;
    onSave({ node: node.trim(), index });
    onClose();
  };

  const handleClose = () => {
    if (canSave) {
      onSave({ node: node.trim(), index });
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
            Get Child At Properties
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
              The variable of the node to get the child from.
            </p>
          </div>

          {/* Index (required, non-negative integer) */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Index (0-based, required):
            </label>
            <Input
              type="number"
              value={String(index)}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                const v = e.target.value.trim();
                if (v === '') {
                  setIndex(NaN as unknown as number);
                  return;
                }
                const n = Number(v);
                setIndex(n);
              }}
              className="w-full"
              placeholder="e.g. 0"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Enter a non-negative integer index. 0 refers to the first child.
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
          Returns the child node at the specified index, or DBNull if the index is out of range.
        </p>
        <p className="text-sm text-gray-700 dark:text-gray-300 mt-2">
          <strong>Parameters:</strong>
        </p>
        <ul className="text-xs text-gray-600 dark:text-gray-400 mt-1 space-y-1">
          <li>1. node (TreeNode, required)</li>
          <li>2. index (number, required, 0-based)</li>
        </ul>
      </div>
    </div>
  );
};
