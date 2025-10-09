import React, { useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface TreeSearchNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: TreeSearchNodeProperties) => void;
  onDelete: () => void;
  initialProperties: TreeSearchNodeProperties;
}

export interface TreeSearchNodeProperties {
  treeVariable: string;  // e.g., users
  fieldName: string;     // e.g., 'name'
  value: string;         // e.g., 'bob'
  operator?: string;     // e.g., 'contains', '>', '<', 'startswith', 'endswith'
  existsOnly?: boolean;  // when true, short-circuit existence check (boolean result)
}

export const TreeSearchNodePropertiesDialog: React.FC<TreeSearchNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [treeVariable, setTreeVariable] = useState(initialProperties.treeVariable || 'tree');
  const [fieldName, setFieldName] = useState(initialProperties.fieldName || 'name');
  const [value, setValue] = useState(initialProperties.value || '');
  const [operator, setOperator] = useState(initialProperties.operator || '');
  const [existsOnly, setExistsOnly] = useState(Boolean(initialProperties.existsOnly) || false);

  const commit = () => {
    onSave({
      treeVariable: treeVariable.trim(),
      fieldName: fieldName.trim(),
      value: value.trim(),
      operator: operator || undefined,
      existsOnly
    });
  };

  const handleSave = () => { commit(); onClose(); };
  const handleClose = () => { commit(); onClose(); };
  const handleDelete = () => { onDelete(); onClose(); };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-md w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Tree Search Properties</h3>
          <button onClick={handleClose} className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200">×</button>
        </div>
          <div className="mb-6">
            <label className="flex items-center space-x-2 text-sm font-medium text-gray-700 dark:text-gray-300">
              <input
                type="checkbox"
                checked={existsOnly}
                onChange={(e) => setExistsOnly(e.target.checked)}
                className="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500 dark:focus:ring-blue-400 dark:bg-gray-700"
              />
              <span>Exists only (short-circuit)</span>
            </label>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1 ml-6">Return true/false as soon as a match is found inside the tree.</p>
          </div>
        <div className="p-6">
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Tree Variable:</label>
            <Input type="text" value={treeVariable} onChange={(e) => setTreeVariable(e.target.value)} placeholder="users" />
          </div>
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Field Name:</label>
            <Input type="text" value={fieldName} onChange={(e) => setFieldName(e.target.value)} placeholder="name" />
          </div>
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Value:</label>
            <Input type="text" value={value} onChange={(e) => setValue(e.target.value)} placeholder="bob" />
          </div>
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Operator (optional):</label>
            <select value={operator} onChange={(e) => setOperator(e.target.value)} className="w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400">
              <option value="">(default)</option>
              <option value="=">=</option>
              <option value=">">&gt;</option>
              <option value="<">&lt;</option>
              <option value=">=">&gt;=</option>
              <option value="<=">&lt;=</option>
              <option value="contains">contains</option>
              <option value="startswith">startswith</option>
              <option value="endswith">endswith</option>
            </select>
          </div>
          <div className="flex gap-3">
            <Button onClick={handleClose} className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200">Save Properties</Button>
            <Button onClick={handleDelete} className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200">Delete</Button>
          </div>
        </div>
      </div>
      <div className="absolute top-1/2 left-1/2 transform translate-x-64 -translate-y-1/2 max-w-sm p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">Search nodes in a tree by field and value with optional operator.</p>
  <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">💡 Examples:<br/>treeSearch(users, 'name', 'bob', 'contains')<br/>treeSearch(products, 'price', 100, '&gt;')</p>
      </div>
    </div>
  );
};
