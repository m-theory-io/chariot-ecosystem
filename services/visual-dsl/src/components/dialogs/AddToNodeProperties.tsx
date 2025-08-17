import React, { useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface AddToNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: AddToNodeProperties) => void;
  onDelete: () => void;
  initialProperties: AddToNodeProperties;
}

export interface AddToNodeProperties {
  collectionName: string;  // Symbolic name of collection to append to
  value: string;           // Value to add to the collection
}

export const AddToNodePropertiesDialog: React.FC<AddToNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [collectionName, setCollectionName] = useState(initialProperties.collectionName || '');
  const [value, setValue] = useState(initialProperties.value || '');

  const handleSave = () => {
    onSave({
      collectionName: collectionName.trim(),
      value: value.trim()
    });
    onClose();
  };

  const handleClose = () => {
    // Save properties when closing
    onSave({
      collectionName: collectionName.trim(),
      value: value.trim()
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
            Add To Collection Properties
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
          {/* Collection Name */}
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Collection Name:
            </label>
            <Input
              type="text"
              value={collectionName}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setCollectionName(e.target.value)}
              className="w-full"
              placeholder="myArray"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Variable name of the collection (unquoted symbol reference)
            </p>
          </div>

          {/* Value */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Value to Add:
            </label>
            <Input
              type="text"
              value={value}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setValue(e.target.value)}
              className="w-full"
              placeholder="'newItem' or variableName or 42"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Value to add - use quotes for strings, no quotes for variables/numbers
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
      <div className="absolute top-1/2 left-1/2 transform translate-x-64 -translate-y-1/2 max-w-sm p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">
          The Add To logicon appends a value to an existing collection (array or map array).
        </p>
        <p className="text-sm text-gray-700 dark:text-gray-300 mt-2">
          <strong>Parameters:</strong>
        </p>
        <ul className="text-xs text-gray-600 dark:text-gray-400 mt-1 space-y-1">
          <li>• Collection Name - Target collection variable</li>
          <li>• Value - Item to append to collection</li>
        </ul>
        <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">
          💡 Examples:<br/>
          addTo(items, 'newItem') - Add string to items collection<br/>
          addTo(numbers, 42) - Add number to numbers collection<br/>
          addTo(userList, userVar) - Add variable to userList collection<br/>
          addTo(myArray, 'string value') - Add quoted string
        </p>
        <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
          <strong>Note:</strong> Collection must exist and be an ArrayValue or []map[string]Value type
        </p>
      </div>
    </div>
  );
};
