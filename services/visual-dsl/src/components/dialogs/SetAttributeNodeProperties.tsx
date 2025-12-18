import React, { useEffect, useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface SetAttributeNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: SetAttributeNodeProperties) => void;
  onDelete: () => void;
  initialProperties: SetAttributeNodeProperties;
}

export interface SetAttributeNodeProperties {
  variableName: string; // node variable
  attributeName: string; // attribute key
  value: string; // attribute value (string literal by default)
}

const SetAttributeNodePropertiesDialog: React.FC<SetAttributeNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [variableName, setVariableName] = useState(initialProperties.variableName || '');
  const [attributeName, setAttributeName] = useState(initialProperties.attributeName || '');
  const [value, setValue] = useState(initialProperties.value || '');
  const [canSave, setCanSave] = useState(false);

  useEffect(() => {
    setCanSave(variableName.trim().length > 0 && attributeName.trim().length > 0 && value.trim().length > 0);
  }, [variableName, attributeName, value]);

  const handleSave = () => {
    if (!canSave) return;
    onSave({ variableName: variableName.trim(), attributeName: attributeName.trim(), value: value.trim() });
    onClose();
  };

  const handleClose = () => {
    if (canSave) {
      onSave({ variableName: variableName.trim(), attributeName: attributeName.trim(), value: value.trim() });
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
            Set Attribute Properties
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
          {/* Variable Name */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Node Variable (required):</label>
            <Input type="text" value={variableName} onChange={(e) => setVariableName(e.target.value)} className="w-full" placeholder="e.g. users" />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">The variable of the node to set the attribute on.</p>
          </div>

          {/* Attribute Name */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Attribute Name (required):</label>
            <Input type="text" value={attributeName} onChange={(e) => setAttributeName(e.target.value)} className="w-full" placeholder="e.g. email" />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Exact key of the attribute to set.</p>
          </div>

          {/* Value */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Value (required):</label>
            <Input type="text" value={value} onChange={(e) => setValue(e.target.value)} className="w-full" placeholder="e.g. 'alice@example.com' or userEmail" />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Enter a string literal or a variable name. Codegen will preserve as-is.</p>
          </div>

          {/* Buttons */}
          <div className="flex gap-3">
            <Button onClick={handleSave} disabled={!canSave} className={`px-6 py-2 ${canSave ? 'bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600' : 'bg-gray-200 dark:bg-gray-600 opacity-60 cursor-not-allowed'} text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200`}>Save Properties</Button>
            <Button onClick={handleDelete} className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200">Delete</Button>
          </div>
        </div>
      </div>

      {/* Help */}
      <div className="absolute top-1/2 left-1/2 transform translate-x-64 -translate-y-1/2 max-w-sm p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">Sets a single attribute on a node.</p>
        <ul className="text-xs text-gray-600 dark:text-gray-400 mt-1 space-y-1">
          <li>1. variableName (TreeNode, required)</li>
          <li>2. attributeName (string, required)</li>
          <li>3. value (any, required)</li>
        </ul>
      </div>
    </div>
  );
};

export default SetAttributeNodePropertiesDialog;
