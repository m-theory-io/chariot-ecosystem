import React, { useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface TreeFindNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: TreeFindNodeProperties) => void;
  onDelete: () => void;
  initialProperties: TreeFindNodeProperties;
}

export interface TreeFindNodeProperties {
  treeVariable?: string; // e.g., usersAgent; optional for implicit runtime search
  fieldName: string;     // e.g., 'id'
  value: string;         // e.g., '123' (string or numeric acceptable)
  operator?: string;     // e.g., '=', 'contains', '>', '<', etc.
  searchAll?: boolean;   // when true, emit implicit form searching runtime trees
}

export const TreeFindNodePropertiesDialog: React.FC<TreeFindNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [treeVariable, setTreeVariable] = useState(initialProperties.treeVariable || '');
  const [fieldName, setFieldName] = useState(initialProperties.fieldName || 'id');
  const [value, setValue] = useState(initialProperties.value || '');
  const [operator, setOperator] = useState(initialProperties.operator || '');
  const [searchAll, setSearchAll] = useState(Boolean(initialProperties.searchAll) || false);

  const commit = () => {
    onSave({
      treeVariable: treeVariable.trim(),
      fieldName: fieldName.trim(),
      value: value.trim(),
      operator: operator || undefined,
      searchAll
    });
  };

  const handleSave = () => { commit(); onClose(); };
  const handleClose = () => { commit(); onClose(); };


  const handleCancel = () => {
    onClose();
  };
  const handleDelete = () => { onDelete(); onClose(); };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
  <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-md w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Tree Find Properties</h3>
          <button onClick={handleCancel} className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200">×</button>
        </div>
        <div className="p-6">
          <div className="mb-4">
            <label className="flex items-center space-x-2 text-sm font-medium text-gray-700 dark:text-gray-300">
              <input
                type="checkbox"
                checked={searchAll}
                onChange={(e) => setSearchAll(e.target.checked)}
                className="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500 dark:focus:ring-blue-400 dark:bg-gray-700"
              />
              <span>Search all runtime trees</span>
            </label>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1 ml-6">When enabled, omits the tree variable and searches across all trees from runtime variables.</p>
          </div>
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Tree Variable:</label>
            {/* Use native input to support disabled prop */}
            <input
              type="text"
              value={treeVariable}
              onChange={(e) => setTreeVariable(e.target.value)}
              placeholder="usersAgent"
              disabled={searchAll}
              className="w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Leave empty to use implicit runtime search.</p>
          </div>
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Field Name:</label>
            <Input type="text" value={fieldName} onChange={(e) => setFieldName(e.target.value)} placeholder="id" />
          </div>
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Value:</label>
            <Input type="text" value={value} onChange={(e) => setValue(e.target.value)} placeholder="123 or 'bob'" />
          </div>
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Operator (optional):</label>
            <select value={operator} onChange={(e) => setOperator(e.target.value)} className="w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400">
              <option value="">(default =)</option>
              <option value="=">=</option>
              <option value=">">&gt;</option>
              <option value="<">&lt;</option>
              <option value=">=">&gt;=</option>
              <option value="<=">&lt;=</option>
              <option value="!=">!=</option>
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
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">Find trees that contain at least one element matching the expression.</p>
  <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">💡 Examples:<br/>treeFind('price', 999, '&gt;')  // implicit runtime search<br/>treeFind(users, 'name', 'bob', 'contains')</p>
      </div>
    </div>
  );
};
