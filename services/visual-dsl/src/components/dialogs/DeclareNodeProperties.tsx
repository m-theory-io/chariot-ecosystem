import React, { useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface DeclareNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: DeclareNodeProperties) => void;
  onDelete: () => void;
  initialProperties: DeclareNodeProperties;
}

export interface DeclareNodeProperties {
  isGlobal: boolean;
  variableName: string;
  typeSpecifier: string;
  initialValue?: string;
}

const CHARIOT_TYPES = [
  { label: 'Any', value: 'V' },
  { label: 'Array', value: 'A' },
  { label: 'Boolean', value: 'L' },
  { label: 'Function', value: 'F' },
  { label: 'HostObject', value: 'O' },
  { label: 'JSONNode', value: 'J' },
  { label: 'MapNode', value: 'M' },
  { label: 'Number', value: 'N' },
  { label: 'Object', value: 'O' },
  { label: 'String', value: 'S' },
  { label: 'Table', value: 'R' },
  { label: 'Tree', value: 'T' },
  { label: 'XMLNode', value: 'X' }
];

export const DeclareNodePropertiesDialog: React.FC<DeclareNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [isGlobal, setIsGlobal] = useState(initialProperties.isGlobal || false);
  const [variableName, setVariableName] = useState(initialProperties.variableName || 'myVar');
  const [typeSpecifier, setTypeSpecifier] = useState(initialProperties.typeSpecifier || 'S');
  const [initialValue, setInitialValue] = useState(initialProperties.initialValue || '');

  const handleSave = () => {
    onSave({
      isGlobal,
      variableName,
      typeSpecifier,
      initialValue: initialValue.trim() || undefined
    });
    onClose();
  };

  const handleClose = () => {
    // Save properties when closing
    onSave({
      isGlobal,
      variableName,
      typeSpecifier,
      initialValue: initialValue.trim() || undefined
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
            Declare Properties
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
          {/* Global Checkbox */}
          <div className="flex items-center space-x-2">
            <input
              type="checkbox"
              id="isGlobal"
              checked={isGlobal}
              onChange={(e) => setIsGlobal(e.target.checked)}
              className="w-4 h-4 text-blue-600 bg-gray-100 border-gray-300 rounded focus:ring-blue-500 dark:focus:ring-blue-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"
            />
            <label htmlFor="isGlobal" className="text-sm font-medium text-gray-700 dark:text-gray-300">
              Global: <span className="text-gray-500 text-xs">({isGlobal ? 'declareGlobal()' : 'declare()'})</span>
            </label>
          </div>

          {/* Variable Name */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Variable Name:
            </label>
            <Input
              type="text"
              value={variableName}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setVariableName(e.target.value)}
              className="w-full"
              placeholder="myVar"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Symbolic name (unquoted)
            </p>
          </div>

          {/* Type Specifier */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Type Specifier:
            </label>
            <select
              value={typeSpecifier}
              onChange={(e) => setTypeSpecifier(e.target.value)}
              className="w-full px-3 py-1 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              {CHARIOT_TYPES.map((type) => (
                <option key={type.value} value={type.value}>
                  {type.label} ({type.value})
                </option>
              ))}
            </select>
          </div>

          {/* Initial Value */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Initial Value: <span className="text-gray-500 text-xs">(optional)</span>
            </label>
            <Input
              type="text"
              value={initialValue}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setInitialValue(e.target.value)}
              className="w-full"
              placeholder="Literal value or leave empty for nested function"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              For complex values, use nested function child nodes
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
      
      {/* Explanatory text */}
      <div className="absolute top-1/2 left-1/2 transform translate-x-64 -translate-y-1/2 max-w-sm p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">
          The Declare logicon creates variable declarations in Chariot. 
          Use <strong>declare()</strong> for local variables or <strong>declareGlobal()</strong> for global variables.
        </p>
        <p className="text-sm text-gray-700 dark:text-gray-300 mt-2">
          <strong>Parameters:</strong>
        </p>
        <ul className="text-xs text-gray-600 dark:text-gray-400 mt-1 space-y-1">
          <li>1. Variable name (symbolic, unquoted)</li>
          <li>2. Type specifier (quoted single character)</li>
          <li>3. Initial value (optional, literal or nested)</li>
        </ul>
      </div>
    </div>
  );
};
