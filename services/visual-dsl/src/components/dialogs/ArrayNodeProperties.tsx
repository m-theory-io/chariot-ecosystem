import React, { useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface ArrayNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: ArrayNodeProperties) => void;
  onDelete: () => void;
  initialProperties: ArrayNodeProperties;
}

export interface ArrayNodeProperties {
  values: string[]; // Array of parameter values
  variableName?: string; // Optional variable name for the result
}

export const ArrayNodePropertiesDialog: React.FC<ArrayNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [values, setValues] = useState<string[]>(initialProperties.values || ['']);
  const [variableName, setVariableName] = useState(initialProperties.variableName || '');

  const handleSave = () => {
    onSave({
      values: values.filter(v => v.trim() !== ''), // Remove empty values
      variableName: variableName.trim() || undefined
    });
    onClose();
  };

  const handleClose = () => {
    // Save properties when closing
    onSave({
      values: values.filter(v => v.trim() !== ''),
      variableName: variableName.trim() || undefined
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

  const addValue = () => {
    setValues([...values, '']);
  };

  const removeValue = (index: number) => {
    if (values.length > 1) {
      setValues(values.filter((_, i) => i !== index));
    }
  };

  const updateValue = (index: number, value: string) => {
    const newValues = [...values];
    newValues[index] = value;
    setValues(newValues);
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-md w-full mx-4">
        {/* Title Bar */}
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Array Properties
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
          {/* Variable Name */}
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Variable Name (optional):
            </label>
            <Input
              type="text"
              value={variableName}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setVariableName(e.target.value)}
              className="w-full"
              placeholder="myArray"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Optional variable to store the array result
            </p>
          </div>

          {/* Array Values */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Array Values:
            </label>
            <div className="space-y-2">
              {values.map((value, index) => (
                <div key={index} className="flex gap-2">
                  <Input
                    type="text"
                    value={value}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => updateValue(index, e.target.value)}
                    className="flex-1"
                    placeholder={`Value ${index + 1}`}
                  />
                  {values.length > 1 && (
                    <button
                      onClick={() => removeValue(index)}
                      className="px-2 py-1 bg-red-100 hover:bg-red-200 dark:bg-red-900 dark:hover:bg-red-800 text-red-600 dark:text-red-400 border border-red-300 dark:border-red-600 rounded text-sm"
                    >
                      Remove
                    </button>
                  )}
                </div>
              ))}
            </div>
            <button
              onClick={addValue}
              className="mt-2 px-3 py-1 bg-blue-100 hover:bg-blue-200 dark:bg-blue-900 dark:hover:bg-blue-800 text-blue-600 dark:text-blue-400 border border-blue-300 dark:border-blue-600 rounded text-sm"
            >
              + Add Value
            </button>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
              Variable list of Chariot.Value parameters for the array() function
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
          The Array logicon creates an ArrayList containing the specified parameters.
        </p>
        <p className="text-sm text-gray-700 dark:text-gray-300 mt-2">
          <strong>Parameters:</strong>
        </p>
        <ul className="text-xs text-gray-600 dark:text-gray-400 mt-1 space-y-1">
          <li>• Variable list of chariot.Value values</li>
          <li>• Each parameter becomes an element in the array</li>
        </ul>
        <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">
          💡 Returns an ArrayList containing all the specified values
        </p>
      </div>
    </div>
  );
};
